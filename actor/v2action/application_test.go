package v2action_test

import (
	"errors"
	"time"

	. "code.cloudfoundry.org/cli/actor/v2action"
	"code.cloudfoundry.org/cli/actor/v2action/v2actionfakes"
	"code.cloudfoundry.org/cli/api/cloudcontroller"
	"code.cloudfoundry.org/cli/api/cloudcontroller/ccv2"

	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Application Actions", func() {
	var (
		actor                     Actor
		fakeCloudControllerClient *v2actionfakes.FakeCloudControllerClient
	)

	BeforeEach(func() {
		fakeCloudControllerClient = new(v2actionfakes.FakeCloudControllerClient)
		actor = NewActor(fakeCloudControllerClient, nil)
	})

	Describe("Application", func() {
		var app Application
		BeforeEach(func() {
			app = Application{}
		})

		Describe("CalculatedBuildpack", func() {
			Context("when buildpack is set", func() {
				BeforeEach(func() {
					app.Buildpack = "foo"
					app.DetectedBuildpack = "bar"
				})

				It("returns back the buildpack", func() {
					Expect(app.CalculatedBuildpack()).To(Equal("foo"))
				})
			})

			Context("only detected buildpack is set", func() {
				BeforeEach(func() {
					app.DetectedBuildpack = "bar"
				})

				It("returns back the detected buildpack", func() {
					Expect(app.CalculatedBuildpack()).To(Equal("bar"))
				})
			})

			Context("neither buildpack or detected buildpack is set", func() {
				It("returns an empty string", func() {
					Expect(app.CalculatedBuildpack()).To(BeEmpty())
				})
			})
		})

		Describe("CalculatedHealthCheckEndpoint", func() {
			var application Application

			Context("when the health check type is http", func() {
				BeforeEach(func() {
					application = Application{
						HealthCheckType:         "http",
						HealthCheckHTTPEndpoint: "/some-endpoint",
					}
				})

				It("returns the endpoint field", func() {
					Expect(application.CalculatedHealthCheckEndpoint()).To(Equal(
						"/some-endpoint"))
				})
			})

			Context("when the health check type is not http", func() {
				BeforeEach(func() {
					application = Application{
						HealthCheckType:         "process",
						HealthCheckHTTPEndpoint: "/some-endpoint",
					}
				})

				It("returns the empty string", func() {
					Expect(application.CalculatedHealthCheckEndpoint()).To(Equal(""))
				})
			})
		})

		Describe("StagingCompleted", func() {
			Context("when staging the application completes", func() {
				It("returns true", func() {
					app.PackageState = ccv2.ApplicationPackageStaged
					Expect(app.StagingCompleted()).To(BeTrue())
				})
			})

			Context("when the application is *not* staged", func() {
				It("returns false", func() {
					app.PackageState = ccv2.ApplicationPackageFailed
					Expect(app.StagingCompleted()).To(BeFalse())
				})
			})
		})

		Describe("StagingFailed", func() {
			Context("when staging the application fails", func() {
				It("returns true", func() {
					app.PackageState = ccv2.ApplicationPackageFailed
					Expect(app.StagingFailed()).To(BeTrue())
				})
			})

			Context("when staging the application does *not* fail", func() {
				It("returns false", func() {
					app.PackageState = ccv2.ApplicationPackageStaged
					Expect(app.StagingFailed()).To(BeFalse())
				})
			})
		})

		Describe("Started", func() {
			Context("when app is started", func() {
				It("returns true", func() {
					Expect(Application{State: ccv2.ApplicationStarted}.Started()).To(BeTrue())
				})
			})

			Context("when app is stopped", func() {
				It("returns false", func() {
					Expect(Application{State: ccv2.ApplicationStopped}.Started()).To(BeFalse())
				})
			})
		})
	})

	Describe("GetApplication", func() {
		Context("when the application exists", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetApplicationReturns(
					ccv2.Application{
						GUID: "some-app-guid",
						Name: "some-app",
					},
					ccv2.Warnings{"foo"},
					nil,
				)
			})

			It("returns the application and warnings", func() {
				app, warnings, err := actor.GetApplication("some-app-guid")
				Expect(err).ToNot(HaveOccurred())
				Expect(app).To(Equal(Application{
					GUID: "some-app-guid",
					Name: "some-app",
				}))
				Expect(warnings).To(Equal(Warnings{"foo"}))

				Expect(fakeCloudControllerClient.GetApplicationCallCount()).To(Equal(1))
				Expect(fakeCloudControllerClient.GetApplicationArgsForCall(0)).To(Equal("some-app-guid"))
			})
		})

		Context("when the application does not exist", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetApplicationReturns(ccv2.Application{}, nil, cloudcontroller.ResourceNotFoundError{})
			})

			It("returns an ApplicationNotFoundError", func() {
				_, _, err := actor.GetApplication("some-app-guid")
				Expect(err).To(MatchError(ApplicationNotFoundError{GUID: "some-app-guid"}))
			})
		})
	})

	Describe("GetApplicationByNameSpace", func() {
		Context("when the application exists", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetApplicationsReturns(
					[]ccv2.Application{
						{
							GUID: "some-app-guid",
							Name: "some-app",
						},
					},
					ccv2.Warnings{"foo"},
					nil,
				)
			})

			It("returns the application and warnings", func() {
				app, warnings, err := actor.GetApplicationByNameAndSpace("some-app", "some-space-guid")
				Expect(err).ToNot(HaveOccurred())
				Expect(app).To(Equal(Application{
					GUID: "some-app-guid",
					Name: "some-app",
				}))
				Expect(warnings).To(Equal(Warnings{"foo"}))

				Expect(fakeCloudControllerClient.GetApplicationsCallCount()).To(Equal(1))
				Expect(fakeCloudControllerClient.GetApplicationsArgsForCall(0)).To(ConsistOf([]ccv2.Query{
					ccv2.Query{
						Filter:   ccv2.NameFilter,
						Operator: ccv2.EqualOperator,
						Value:    "some-app",
					},
					ccv2.Query{
						Filter:   ccv2.SpaceGUIDFilter,
						Operator: ccv2.EqualOperator,
						Value:    "some-space-guid",
					},
				}))
			})
		})

		Context("when the application does not exists", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetApplicationsReturns([]ccv2.Application{}, nil, nil)
			})

			It("returns an ApplicationNotFoundError", func() {
				_, _, err := actor.GetApplicationByNameAndSpace("some-app", "some-space-guid")
				Expect(err).To(MatchError(ApplicationNotFoundError{Name: "some-app"}))
			})
		})

		Context("when the cloud controller client returns an error", func() {
			var expectedError error

			BeforeEach(func() {
				expectedError = errors.New("I am a CloudControllerClient Error")
				fakeCloudControllerClient.GetApplicationsReturns([]ccv2.Application{}, nil, expectedError)
			})

			It("returns the error", func() {
				_, _, err := actor.GetApplicationByNameAndSpace("some-app", "some-space-guid")
				Expect(err).To(MatchError(expectedError))
			})
		})
	})

	Describe("GetRouteApplications", func() {
		Context("when the CC client returns no errors", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetRouteApplicationsReturns(
					[]ccv2.Application{
						{
							GUID: "application-guid",
							Name: "application-name",
						},
					}, ccv2.Warnings{"route-applications-warning"}, nil)
			})
			It("returns the applications bound to the route and warnings", func() {
				applications, warnings, err := actor.GetRouteApplications("route-guid", nil)
				Expect(fakeCloudControllerClient.GetRouteApplicationsCallCount()).To(Equal(1))
				Expect(fakeCloudControllerClient.GetRouteApplicationsArgsForCall(0)).To(Equal("route-guid"))

				Expect(err).ToNot(HaveOccurred())
				Expect(warnings).To(ConsistOf("route-applications-warning"))
				Expect(applications).To(ConsistOf(
					Application{
						GUID: "application-guid",
						Name: "application-name",
					},
				))
			})
		})

		Context("when the CC client returns an error", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetRouteApplicationsReturns(
					[]ccv2.Application{}, ccv2.Warnings{"route-applications-warning"}, errors.New("get-route-applications-error"))
			})

			It("returns the error and warnings", func() {
				apps, warnings, err := actor.GetRouteApplications("route-guid", nil)
				Expect(fakeCloudControllerClient.GetRouteApplicationsCallCount()).To(Equal(1))
				Expect(fakeCloudControllerClient.GetRouteApplicationsArgsForCall(0)).To(Equal("route-guid"))

				Expect(err).To(MatchError("get-route-applications-error"))
				Expect(warnings).To(ConsistOf("route-applications-warning"))
				Expect(apps).To(BeNil())
			})
		})

		Context("when a query parameter exists", func() {
			It("passes the query to the client", func() {
				expectedQuery := []ccv2.Query{
					{
						Filter:   ccv2.RouteGUIDFilter,
						Operator: ccv2.EqualOperator,
						Value:    "route-guid",
					}}

				_, _, err := actor.GetRouteApplications("route-guid", expectedQuery)
				Expect(err).ToNot(HaveOccurred())
				_, query := fakeCloudControllerClient.GetRouteApplicationsArgsForCall(0)
				Expect(query).To(Equal(expectedQuery))
			})
		})
	})

	Describe("StartApplication", func() {
		var (
			app            Application
			fakeNOAAClient *v2actionfakes.FakeNOAAClient
			fakeConfig     *v2actionfakes.FakeConfig

			messages <-chan *LogMessage
			logErrs  <-chan error
			warnings <-chan string
			errs     <-chan error

			eventStream chan *events.LogMessage
			errStream   chan error
		)

		BeforeEach(func() {
			fakeConfig = new(v2actionfakes.FakeConfig)
			fakeConfig.StagingTimeoutReturns(time.Minute)
			fakeConfig.StartupTimeoutReturns(time.Minute)

			app = Application{
				GUID:      "some-app-guid",
				Name:      "some-app",
				Instances: 0,
			}

			fakeNOAAClient = new(v2actionfakes.FakeNOAAClient)
			fakeNOAAClient.TailingLogsStub = func(_ string, _ string) (<-chan *events.LogMessage, <-chan error) {
				eventStream = make(chan *events.LogMessage)
				errStream = make(chan error)
				return eventStream, errStream
			}
			fakeNOAAClient.CloseStub = func() error {
				close(errStream)
				close(eventStream)
				return nil
			}

			fakeCloudControllerClient.UpdateApplicationReturns(ccv2.Application{GUID: "some-app-guid",
				Instances: 0,
				Name:      "some-app",
			}, ccv2.Warnings{"update-warning"}, nil)

			appCount := 0
			fakeCloudControllerClient.GetApplicationStub = func(appGUID string) (ccv2.Application, ccv2.Warnings, error) {
				if appCount == 0 {
					appCount += 1
					return ccv2.Application{
						GUID:         "some-app-guid",
						Instances:    0,
						Name:         "some-app",
						PackageState: ccv2.ApplicationPackagePending,
					}, ccv2.Warnings{"app-warnings-1"}, nil
				}

				return ccv2.Application{
					GUID:         "some-app-guid",
					Name:         "some-app",
					Instances:    2,
					PackageState: ccv2.ApplicationPackageStaged,
				}, ccv2.Warnings{"app-warnings-2"}, nil
			}

			instanceCount := 0
			fakeCloudControllerClient.GetApplicationInstancesByApplicationStub = func(guid string) (map[int]ccv2.ApplicationInstance, ccv2.Warnings, error) {
				if instanceCount == 0 {
					instanceCount += 1
					return map[int]ccv2.ApplicationInstance{
						0: {State: ccv2.ApplicationInstanceStarting},
						1: {State: ccv2.ApplicationInstanceStarting},
					}, ccv2.Warnings{"app-instance-warnings-1"}, nil
				}

				return map[int]ccv2.ApplicationInstance{
					0: {State: ccv2.ApplicationInstanceStarting},
					1: {State: ccv2.ApplicationInstanceRunning},
				}, ccv2.Warnings{"app-instance-warnings-2"}, nil
			}
		})

		AfterEach(func() {
			Eventually(fakeNOAAClient.CloseCallCount).Should(Equal(1))

			Eventually(messages).Should(BeClosed())
			Eventually(logErrs).Should(BeClosed())
			Eventually(warnings).Should(BeClosed())
			Eventually(errs).Should(BeClosed())
		})

		It("starts and polls the app instance", func() {
			messages, logErrs, warnings, errs = actor.StartApplication(app, fakeNOAAClient, fakeConfig)

			Eventually(<-warnings).Should(Equal("update-warning"))
			Eventually(<-warnings).Should(Equal("app-warnings-1"))
			Eventually(<-warnings).Should(Equal("app-warnings-2"))
			Eventually(<-warnings).Should(Equal("app-instance-warnings-1"))
			Eventually(<-warnings).Should(Equal("app-instance-warnings-2"))

			Expect(fakeConfig.PollingIntervalCallCount()).To(Equal(2))

			Expect(fakeCloudControllerClient.UpdateApplicationCallCount()).To(Equal(1))
			app := fakeCloudControllerClient.UpdateApplicationArgsForCall(0)
			Expect(app).To(Equal(ccv2.Application{
				GUID:  "some-app-guid",
				State: ccv2.ApplicationStarted,
			}))

			Expect(fakeCloudControllerClient.GetApplicationCallCount()).To(Equal(2))
			Expect(fakeCloudControllerClient.GetApplicationInstancesByApplicationCallCount()).To(Equal(2))
		})

		Context("when updating the application fails", func() {
			var expectedErr error
			BeforeEach(func() {
				expectedErr = errors.New("I am a banana!!!!")
				fakeCloudControllerClient.UpdateApplicationReturns(ccv2.Application{}, ccv2.Warnings{"update-warning"}, expectedErr)
			})

			It("sends the update error and never polls", func() {
				messages, logErrs, warnings, errs = actor.StartApplication(app, fakeNOAAClient, fakeConfig)

				Eventually(<-warnings).Should(Equal("update-warning"))
				Eventually(<-errs).Should(MatchError(expectedErr))

				Expect(fakeConfig.PollingIntervalCallCount()).To(Equal(0))
				Expect(fakeCloudControllerClient.GetApplicationCallCount()).To(Equal(0))
				Expect(fakeCloudControllerClient.GetApplicationInstancesByApplicationCallCount()).To(Equal(0))
			})
		})

		Context("staging issues", func() {
			Context("when polling fails", func() {
				var expectedErr error
				BeforeEach(func() {
					expectedErr = errors.New("I am a banana!!!!")
					fakeCloudControllerClient.GetApplicationStub = func(appGUID string) (ccv2.Application, ccv2.Warnings, error) {
						return ccv2.Application{}, ccv2.Warnings{"app-warnings-1"}, expectedErr
					}
				})

				It("sends the error and stops polling", func() {
					messages, logErrs, warnings, errs = actor.StartApplication(app, fakeNOAAClient, fakeConfig)

					Eventually(<-warnings).Should(Equal("update-warning"))
					Eventually(<-warnings).Should(Equal("app-warnings-1"))
					Eventually(<-errs).Should(MatchError(expectedErr))

					Expect(fakeConfig.PollingIntervalCallCount()).To(Equal(0))
					Expect(fakeCloudControllerClient.GetApplicationCallCount()).To(Equal(1))
					Expect(fakeCloudControllerClient.GetApplicationInstancesByApplicationCallCount()).To(Equal(0))
				})
			})

			Context("when the application fails to stage", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetApplicationStub = func(appGUID string) (ccv2.Application, ccv2.Warnings, error) {
						return ccv2.Application{
							GUID:                "some-app-guid",
							Name:                "some-app",
							Instances:           2,
							PackageState:        ccv2.ApplicationPackageFailed,
							StagingFailedReason: "OhNoes",
						}, ccv2.Warnings{"app-warnings-1"}, nil
					}
				})

				It("sends a staging error and stops polling", func() {
					messages, logErrs, warnings, errs = actor.StartApplication(app, fakeNOAAClient, fakeConfig)

					Eventually(<-warnings).Should(Equal("update-warning"))
					Eventually(<-warnings).Should(Equal("app-warnings-1"))
					Eventually(<-errs).Should(MatchError(StagingFailedError{Reason: "OhNoes"}))

					Expect(fakeConfig.PollingIntervalCallCount()).To(Equal(0))
					Expect(fakeConfig.StagingTimeoutCallCount()).To(Equal(1))
					Expect(fakeCloudControllerClient.GetApplicationCallCount()).To(Equal(1))
					Expect(fakeCloudControllerClient.GetApplicationInstancesByApplicationCallCount()).To(Equal(0))
				})
			})

			Context("when the application takes too long to stage", func() {
				BeforeEach(func() {
					fakeConfig.StagingTimeoutReturns(0)
				})

				It("sends a timeout error and stops polling", func() {
					messages, logErrs, warnings, errs = actor.StartApplication(app, fakeNOAAClient, fakeConfig)

					Eventually(<-warnings).Should(Equal("update-warning"))
					Eventually(<-errs).Should(MatchError(StagingTimeoutError{Name: "some-app"}))

					Expect(fakeConfig.PollingIntervalCallCount()).To(Equal(0))
					Expect(fakeConfig.StagingTimeoutCallCount()).To(Equal(1))
					Expect(fakeCloudControllerClient.GetApplicationCallCount()).To(Equal(0))
					Expect(fakeCloudControllerClient.GetApplicationInstancesByApplicationCallCount()).To(Equal(0))
				})
			})
		})

		Context("starting issues", func() {
			Context("when polling fails", func() {
				var expectedErr error
				BeforeEach(func() {
					expectedErr = errors.New("I am a banana!!!!")
					fakeCloudControllerClient.GetApplicationInstancesByApplicationStub = func(guid string) (map[int]ccv2.ApplicationInstance, ccv2.Warnings, error) {
						return nil, ccv2.Warnings{"app-instance-warnings-1"}, expectedErr
					}
				})

				It("sends the error and stops polling", func() {
					messages, logErrs, warnings, errs = actor.StartApplication(app, fakeNOAAClient, fakeConfig)

					Eventually(<-warnings).Should(Equal("update-warning"))
					Eventually(<-warnings).Should(Equal("app-warnings-1"))
					Eventually(<-warnings).Should(Equal("app-warnings-2"))
					Eventually(<-warnings).Should(Equal("app-instance-warnings-1"))
					Eventually(<-errs).Should(MatchError(expectedErr))

					Expect(fakeConfig.PollingIntervalCallCount()).To(Equal(1))
					Expect(fakeCloudControllerClient.GetApplicationInstancesByApplicationCallCount()).To(Equal(1))
				})
			})

			Context("when the application takes too long to start", func() {
				BeforeEach(func() {
					fakeConfig.StartupTimeoutReturns(0)
				})

				It("sends a timeout error and stops polling", func() {
					messages, logErrs, warnings, errs = actor.StartApplication(app, fakeNOAAClient, fakeConfig)

					Eventually(<-warnings).Should(Equal("update-warning"))
					Eventually(<-warnings).Should(Equal("app-warnings-1"))
					Eventually(<-warnings).Should(Equal("app-warnings-2"))
					Eventually(<-errs).Should(MatchError(StartupTimeoutError{Name: "some-app"}))

					Expect(fakeConfig.PollingIntervalCallCount()).To(Equal(1))
					Expect(fakeConfig.StartupTimeoutCallCount()).To(Equal(1))
					Expect(fakeCloudControllerClient.GetApplicationInstancesByApplicationCallCount()).To(Equal(0))
				})
			})

			Context("when the application crashes", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetApplicationInstancesByApplicationStub = func(guid string) (map[int]ccv2.ApplicationInstance, ccv2.Warnings, error) {
						return map[int]ccv2.ApplicationInstance{
							0: {State: ccv2.ApplicationInstanceCrashed},
						}, ccv2.Warnings{"app-instance-warnings-1"}, nil
					}
				})

				It("returns an ApplicationInstanceCrashedError and stops polling", func() {
					messages, logErrs, warnings, errs = actor.StartApplication(app, fakeNOAAClient, fakeConfig)

					Eventually(<-warnings).Should(Equal("update-warning"))
					Eventually(<-warnings).Should(Equal("app-warnings-1"))
					Eventually(<-warnings).Should(Equal("app-warnings-2"))
					Eventually(<-warnings).Should(Equal("app-instance-warnings-1"))
					Eventually(<-errs).Should(MatchError(ApplicationInstanceCrashedError{Name: "some-app"}))

					Expect(fakeConfig.PollingIntervalCallCount()).To(Equal(1))
					Expect(fakeConfig.StartupTimeoutCallCount()).To(Equal(1))
					Expect(fakeCloudControllerClient.GetApplicationInstancesByApplicationCallCount()).To(Equal(1))
				})
			})

			Context("when the application flaps", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetApplicationInstancesByApplicationStub = func(guid string) (map[int]ccv2.ApplicationInstance, ccv2.Warnings, error) {
						return map[int]ccv2.ApplicationInstance{
							0: {State: ccv2.ApplicationInstanceFlapping},
						}, ccv2.Warnings{"app-instance-warnings-1"}, nil
					}
				})

				It("returns an ApplicationInstanceFlappingError and stops polling", func() {
					messages, logErrs, warnings, errs = actor.StartApplication(app, fakeNOAAClient, fakeConfig)

					Eventually(<-warnings).Should(Equal("update-warning"))
					Eventually(<-warnings).Should(Equal("app-warnings-1"))
					Eventually(<-warnings).Should(Equal("app-warnings-2"))
					Eventually(<-warnings).Should(Equal("app-instance-warnings-1"))
					Eventually(<-errs).Should(MatchError(ApplicationInstanceFlappingError{Name: "some-app"}))

					Expect(fakeConfig.PollingIntervalCallCount()).To(Equal(1))
					Expect(fakeConfig.StartupTimeoutCallCount()).To(Equal(1))
					Expect(fakeCloudControllerClient.GetApplicationInstancesByApplicationCallCount()).To(Equal(1))
				})
			})
		})
	})

	Describe("SetApplicationHealthCheckTypeByNameAndSpace", func() {
		Context("when setting an http endpoint with a health check that is not http", func() {
			It("returns an http health check invalid error", func() {
				_, _, err := actor.SetApplicationHealthCheckTypeByNameAndSpace(
					"some-app", "some-space-guid", "some-health-check-type", "/foo")
				Expect(err).To(MatchError(HTTPHealthCheckInvalidError{}))
			})
		})

		Context("when the app exists", func() {
			Context("when the desired health check type is different", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetApplicationsReturns(
						[]ccv2.Application{
							{GUID: "some-app-guid"},
						},
						ccv2.Warnings{"get application warning"},
						nil,
					)
					fakeCloudControllerClient.UpdateApplicationReturns(
						ccv2.Application{
							GUID:            "some-app-guid",
							HealthCheckType: "process",
						},
						ccv2.Warnings{"update warnings"},
						nil,
					)
				})

				It("sets the desired health check type and returns the warnings", func() {
					returnedApp, warnings, err := actor.SetApplicationHealthCheckTypeByNameAndSpace(
						"some-app", "some-space-guid", "process", "/")
					Expect(err).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("get application warning", "update warnings"))

					Expect(returnedApp).To(Equal(Application{
						GUID:            "some-app-guid",
						HealthCheckType: "process",
					}))

					Expect(fakeCloudControllerClient.UpdateApplicationCallCount()).To(Equal(1))
					app := fakeCloudControllerClient.UpdateApplicationArgsForCall(0)
					Expect(app).To(Equal(ccv2.Application{
						GUID:            "some-app-guid",
						HealthCheckType: "process",
					}))
				})
			})

			Context("when the desired health check type is 'http'", func() {
				Context("when the desired http endpoint is already set", func() {
					BeforeEach(func() {
						fakeCloudControllerClient.GetApplicationsReturns(
							[]ccv2.Application{
								{GUID: "some-app-guid", HealthCheckType: "http", HealthCheckHTTPEndpoint: "/"},
							},
							ccv2.Warnings{"get application warning"},
							nil,
						)
					})

					It("does not send the update", func() {
						_, warnings, err := actor.SetApplicationHealthCheckTypeByNameAndSpace(
							"some-app", "some-space-guid", "http", "/")
						Expect(err).ToNot(HaveOccurred())
						Expect(warnings).To(ConsistOf("get application warning"))

						Expect(fakeCloudControllerClient.UpdateApplicationCallCount()).To(Equal(0))
					})
				})

				Context("when the desired http endpoint is not set", func() {
					BeforeEach(func() {
						fakeCloudControllerClient.GetApplicationsReturns(
							[]ccv2.Application{
								{GUID: "some-app-guid", HealthCheckType: "http", HealthCheckHTTPEndpoint: "/"},
							},
							ccv2.Warnings{"get application warning"},
							nil,
						)
						fakeCloudControllerClient.UpdateApplicationReturns(
							ccv2.Application{},
							ccv2.Warnings{"update warnings"},
							nil,
						)
					})

					It("sets the desired health check type and returns the warnings", func() {
						_, warnings, err := actor.SetApplicationHealthCheckTypeByNameAndSpace(
							"some-app", "some-space-guid", "http", "/v2/anything")
						Expect(err).ToNot(HaveOccurred())

						Expect(fakeCloudControllerClient.UpdateApplicationCallCount()).To(Equal(1))
						app := fakeCloudControllerClient.UpdateApplicationArgsForCall(0)
						Expect(app).To(Equal(ccv2.Application{
							GUID:                    "some-app-guid",
							HealthCheckType:         "http",
							HealthCheckHTTPEndpoint: "/v2/anything",
						}))

						Expect(warnings).To(ConsistOf("get application warning", "update warnings"))
					})
				})
			})

			Context("when the application health check type is already set to the desired type", func() {
				BeforeEach(func() {
					fakeCloudControllerClient.GetApplicationsReturns(
						[]ccv2.Application{
							{
								GUID:            "some-app-guid",
								HealthCheckType: "process",
							},
						},
						ccv2.Warnings{"get application warning"},
						nil,
					)
				})

				It("does not update the health check type", func() {
					returnedApp, warnings, err := actor.SetApplicationHealthCheckTypeByNameAndSpace(
						"some-app", "some-space-guid", "process", "/")
					Expect(err).ToNot(HaveOccurred())
					Expect(warnings).To(ConsistOf("get application warning"))
					Expect(returnedApp).To(Equal(Application{
						GUID:            "some-app-guid",
						HealthCheckType: "process",
					}))

					Expect(fakeCloudControllerClient.UpdateApplicationCallCount()).To(Equal(0))
				})
			})
		})

		Context("when getting the application returns an error", func() {
			BeforeEach(func() {
				fakeCloudControllerClient.GetApplicationsReturns(
					[]ccv2.Application{}, ccv2.Warnings{"get application warning"}, errors.New("get application error"))
			})

			It("returns the error and warnings", func() {
				_, warnings, err := actor.SetApplicationHealthCheckTypeByNameAndSpace(
					"some-app", "some-space-guid", "process", "/")

				Expect(warnings).To(ConsistOf("get application warning"))
				Expect(err).To(MatchError("get application error"))
			})
		})

		Context("when updating the application returns an error", func() {
			var expectedErr error

			BeforeEach(func() {
				expectedErr = errors.New("foo bar")
				fakeCloudControllerClient.GetApplicationsReturns(
					[]ccv2.Application{
						{GUID: "some-app-guid"},
					},
					ccv2.Warnings{"get application warning"},
					nil,
				)
				fakeCloudControllerClient.UpdateApplicationReturns(
					ccv2.Application{},
					ccv2.Warnings{"update warnings"},
					expectedErr,
				)
			})

			It("returns the error and warnings", func() {
				_, warnings, err := actor.SetApplicationHealthCheckTypeByNameAndSpace(
					"some-app", "some-space-guid", "process", "/")
				Expect(err).To(MatchError(expectedErr))
				Expect(warnings).To(ConsistOf("get application warning", "update warnings"))
			})
		})
	})
})

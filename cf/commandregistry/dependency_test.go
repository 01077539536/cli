package commandregistry_test

import (
	"github.com/cloudfoundry/cli/cf/commandregistry"
	"github.com/cloudfoundry/cli/cf/trace/tracefakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dependency", func() {
	var dependency commandregistry.Dependency

	It("populates all fields by calling all the dependency contructors", func() {
		fakeLogger := new(tracefakes.FakePrinter)
		dependency = commandregistry.NewDependency(fakeLogger)

		Expect(dependency.Ui).ToNot(BeNil())
		Expect(dependency.Config).ToNot(BeNil())
		Expect(dependency.RepoLocator).ToNot(BeNil())
		Expect(dependency.PluginConfig).ToNot(BeNil())
		Expect(dependency.ManifestRepo).ToNot(BeNil())
		Expect(dependency.AppManifest).ToNot(BeNil())
		Expect(dependency.Gateways).ToNot(BeNil())
		Expect(dependency.TeePrinter).ToNot(BeNil())
		Expect(dependency.PluginRepo).ToNot(BeNil())
		Expect(dependency.PluginModels).ToNot(BeNil())
		Expect(dependency.ServiceBuilder).ToNot(BeNil())
		Expect(dependency.BrokerBuilder).ToNot(BeNil())
		Expect(dependency.PlanBuilder).ToNot(BeNil())
		Expect(dependency.ServiceHandler).ToNot(BeNil())
		Expect(dependency.ServicePlanHandler).ToNot(BeNil())
		Expect(dependency.WordGenerator).ToNot(BeNil())
		Expect(dependency.AppZipper).ToNot(BeNil())
		Expect(dependency.AppFiles).ToNot(BeNil())
		Expect(dependency.PushActor).ToNot(BeNil())
		Expect(dependency.ChecksumUtil).ToNot(BeNil())
	})
})
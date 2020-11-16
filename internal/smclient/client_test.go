package smclient

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

var _ = Describe("Utils test", func() {

	It("should succeed", func() {
		cl, _ := NewClient("pique", &ClientConfig{
			ClientID:     "sb-870fbfb3-bbd9-4940-9bf3-87f713fd917f!b3072|service-manager!b3",
			ClientSecret: "MKFM4YdiXKjaTArNzmK+9ryvDAY=",
			URL:          "https://service-manager.cfapps.stagingaws.hanavlab.ondemand.com",
			SSLDisabled:  false,
		})

		i, err := cl.ListInstances(nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(i).To(Not(BeNil()))
	})

	FIt("", func() {
		cl, _ := NewClient("MyDemoSubaccount0909", &ClientConfig{
			ClientID:     "sb-9249744f-ff1c-4cba-a940-c3284b53c77d!b15166|service-manager!b4065",
			ClientSecret: "F6SPiLObbP14ruR+Qtb6xwiDsFk=",
			URL:          "https://service-manager.cfapps.sap.hana.ondemand.com",
			SSLDisabled:  false,
		})

		parameters := Parameters{
			FieldQuery: []string{
				fmt.Sprintf("name eq '%s'", "eyaln-test-k8s-bindings-13"),
				fmt.Sprintf("service_instance_id eq '%s'", "16561cf0-afe8-47de-a23f-9f852d08013c")},
			LabelQuery: []string{
				fmt.Sprintf("_clusterid eq '%s'", "some-cluster-id"),
				fmt.Sprintf("_namespace eq '%s'", "default"),
				fmt.Sprintf("_k8sname eq '%s'", "eyaln-test-k8s-bindings-13"),
			},
			GeneralParams: []string{"attach_last_operations=true"},
		}

		bindings, _ := cl.ListBindings(&parameters)
		Expect(bindings.ServiceBindings[0].ID).To(Not(BeNil()))
	})
})

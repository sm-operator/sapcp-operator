package test

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sm-operator/sapcp-operator/internal/smclient"
	"github.com/sm-operator/sapcp-operator/internal/smclient/types"
	"net/http"
)

var (
	instance    *types.ServiceInstance
	serviceID   = "service_id"
	serviceName = "mongo"
	planName    = "small"
	planID      = "service_plan_id"
)

var _ = Describe("Instance test", func() {

	BeforeEach(func() {
		instance = &types.ServiceInstance{
			ID:            "instanceID",
			Name:          "instance1",
			ServicePlanID: planID,
			PlatformID:    "platform_id",
			Context:       json.RawMessage("{}"),
		}
		instancesArray := []types.ServiceInstance{*instance}
		instances := types.ServiceInstances{ServiceInstances: instancesArray}
		responseBody, _ := json.Marshal(instances)

		handlerDetails = []HandlerDetails{
			{Method: http.MethodGet, Path: web.ServiceInstancesURL, ResponseBody: responseBody, ResponseStatusCode: http.StatusOK},
		}
	})

	Describe("List service instances", func() {
		Context("When there are service instances registered", func() {
			It("should return all", func() {
				result, err := client.ListInstances(params)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(result.ServiceInstances).To(HaveLen(1))
				Expect(result.ServiceInstances[0]).To(Equal(*instance))
			})
		})

		Context("When there are no service instances registered", func() {
			BeforeEach(func() {
				var instancesArray []types.ServiceInstance
				instances := types.ServiceInstances{ServiceInstances: instancesArray}
				responseBody, _ := json.Marshal(instances)

				handlerDetails = []HandlerDetails{
					{Method: http.MethodGet, Path: web.ServiceInstancesURL, ResponseBody: responseBody, ResponseStatusCode: http.StatusOK},
				}
			})
			It("should return an empty array", func() {
				result, err := client.ListInstances(params)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(result.ServiceInstances).To(HaveLen(0))
			})
		})

		Context("When invalid status code is returned", func() {
			BeforeEach(func() {
				handlerDetails = []HandlerDetails{
					{Method: http.MethodGet, Path: web.ServiceInstancesURL, ResponseStatusCode: http.StatusCreated},
				}
			})
			It("should handle status code != 200", func() {
				_, err := client.ListInstances(params)
				Expect(err).Should(HaveOccurred())
				verifyErrorMsg(err.Error(), handlerDetails[0].Path, handlerDetails[0].ResponseBody, handlerDetails[0].ResponseStatusCode)
			})
		})

		Context("When invalid status code is returned", func() {
			BeforeEach(func() {
				handlerDetails = []HandlerDetails{
					{Method: http.MethodGet, Path: web.ServiceInstancesURL, ResponseStatusCode: http.StatusBadRequest},
				}
			})
			It("should handle status code > 299", func() {
				_, err := client.ListInstances(params)
				Expect(err).Should(HaveOccurred())
				verifyErrorMsg(err.Error(), handlerDetails[0].Path, handlerDetails[0].ResponseBody, handlerDetails[0].ResponseStatusCode)
			})
		})
	})

	Describe("Get service instance", func() {
		Context("When there is instance with this id", func() {
			BeforeEach(func() {
				responseBody, _ := json.Marshal(instance)
				handlerDetails = []HandlerDetails{
					{Method: http.MethodGet, Path: web.ServiceInstancesURL + "/", ResponseBody: responseBody, ResponseStatusCode: http.StatusOK},
				}
			})
			It("should return it", func() {
				result, err := client.GetInstanceByID(instance.ID, params)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(result).To(Equal(instance))
			})
		})

		Context("When there is no instance with this id", func() {
			BeforeEach(func() {
				handlerDetails = []HandlerDetails{
					{Method: http.MethodGet, Path: web.ServiceInstancesURL + "/", ResponseStatusCode: http.StatusNotFound},
				}
			})
			It("should return 404", func() {
				_, err := client.GetInstanceByID(instance.ID, params)
				Expect(err).Should(HaveOccurred())
				verifyErrorMsg(err.Error(), handlerDetails[0].Path+instance.ID, handlerDetails[0].ResponseBody, handlerDetails[0].ResponseStatusCode)
			})
		})

		Context("When invalid status code is returned", func() {
			BeforeEach(func() {
				handlerDetails = []HandlerDetails{
					{Method: http.MethodGet, Path: web.ServiceInstancesURL + "/", ResponseStatusCode: http.StatusCreated},
				}
			})
			It("should handle status code != 200", func() {
				_, err := client.GetInstanceByID(instance.ID, params)
				Expect(err).Should(HaveOccurred())
				verifyErrorMsg(err.Error(), handlerDetails[0].Path+instance.ID, handlerDetails[0].ResponseBody, handlerDetails[0].ResponseStatusCode)
			})
		})

		Context("When invalid status code is returned", func() {
			BeforeEach(func() {
				handlerDetails = []HandlerDetails{
					{Method: http.MethodGet, Path: web.ServiceInstancesURL + "/", ResponseStatusCode: http.StatusBadRequest},
				}
			})

			It("should handle status code > 299", func() {
				_, err := client.GetInstanceByID(instance.ID, params)
				Expect(err).Should(HaveOccurred())
				verifyErrorMsg(err.Error(), handlerDetails[0].Path+instance.ID, handlerDetails[0].ResponseBody, handlerDetails[0].ResponseStatusCode)

			})
		})
	})

	Describe("Provision", func() {
		BeforeEach(func() {
			instanceResponseBody, _ := json.Marshal(instance)
			offering := &types.ServiceOffering{
				ID:          "service_id",
				Name:        serviceName,
				CatalogName: serviceName,
			}
			plan := &types.ServicePlan{
				ID:                planID,
				Name:              planName,
				CatalogName:       serviceName,
				ServiceOfferingID: serviceID,
			}

			plansArray := []types.ServicePlan{*plan}
			plans := types.ServicePlans{ServicePlans: plansArray}
			plansBody, _ := json.Marshal(plans)

			offeringArray := []types.ServiceOffering{*offering}
			offerings := types.ServiceOfferings{ServiceOfferings: offeringArray}
			offeringResponseBody, _ := json.Marshal(offerings)
			handlerDetails = []HandlerDetails{
				{Method: http.MethodPost, Path: web.ServiceInstancesURL, ResponseBody: instanceResponseBody, ResponseStatusCode: http.StatusCreated},
				{Method: http.MethodGet, Path: web.ServiceOfferingsURL, ResponseBody: offeringResponseBody, ResponseStatusCode: http.StatusOK},
				{Method: http.MethodGet, Path: web.ServicePlansURL, ResponseBody: plansBody, ResponseStatusCode: http.StatusOK},
			}
		})

		Context("When valid instance is being provisioned synchronously", func() {
			It("should provision successfully", func() {
				responseInstanceID, location, err := client.Provision(instance, serviceName, planName, params)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(location).Should(HaveLen(0))
				Expect(responseInstanceID).To(Equal(instance.ID))
			})

			Context("When multiple matching plan names returned from SM", func() {
				BeforeEach(func() {
					plan1 := &types.ServicePlan{
						ID:                planID,
						Name:              planName,
						CatalogName:       serviceName,
						ServiceOfferingID: serviceID,
					}
					plan2 := &types.ServicePlan{
						ID:                "planID2",
						Name:              planName,
						CatalogName:       serviceName,
						ServiceOfferingID: serviceID,
					}

					plansArray := []types.ServicePlan{*plan1, *plan2}
					plans := types.ServicePlans{ServicePlans: plansArray}
					plansBody, _ := json.Marshal(plans)
					handlerDetails[2] = HandlerDetails{Method: http.MethodGet, Path: web.ServicePlansURL, ResponseBody: plansBody, ResponseStatusCode: http.StatusOK}
				})

				It("should provision successfully", func() {
					responseInstanceID, location, err := client.Provision(instance, "mongo", "small", params)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(location).Should(HaveLen(0))
					Expect(responseInstanceID).To(Equal(instance.ID))
				})
			})

			Context("When no plan id provided", func() {
				It("should provision successfully", func() {
					instance.ServicePlanID = ""
					responseInstanceID, location, err := client.Provision(instance, "mongo", "small", params)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(location).Should(HaveLen(0))
					Expect(responseInstanceID).To(Equal(instance.ID))
				})
			})
		})

		Context("When invalid instance is being provisioned synchronously", func() {
			When("No service name", func() {
				It("should fail to provision", func() {
					responseInstanceID, location, err := client.Provision(instance, "", "small", params)
					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())
					Expect(responseInstanceID).Should(BeEmpty())
				})
			})

			When("No plan name", func() {
				It("should fail to provision", func() {
					responseInstanceID, location, err := client.Provision(instance, "mongo", "", params)
					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())
					Expect(responseInstanceID).Should(BeEmpty())
				})
			})

			When("Plan id not match plan name", func() {
				It("should fail", func() {
					instance.ServicePlanID = "some-id"
					responseInstanceID, location, err := client.Provision(instance, "mongo", "small", params)
					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())
					Expect(responseInstanceID).Should(BeEmpty())
				})
			})

			When("Service not exist", func() {
				BeforeEach(func() {
					var offeringsArray []types.ServiceOffering
					offerings := types.ServiceOfferings{ServiceOfferings: offeringsArray}
					responseBody, _ := json.Marshal(offerings)
					handlerDetails[1] = HandlerDetails{Method: http.MethodGet, Path: web.ServiceOfferingsURL, ResponseBody: responseBody, ResponseStatusCode: http.StatusOK}
				})
				It("should fail", func() {
					responseInstanceID, location, err := client.Provision(instance, "mongo2", "small", params)
					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())
					Expect(responseInstanceID).Should(BeEmpty())
				})
			})
		})

		Context("When valid instance is being provisioned asynchronously", func() {
			var locationHeader string
			BeforeEach(func() {
				locationHeader = "/v1/service_instances/12345/operations/1234"
				handlerDetails[0] = HandlerDetails{Method: http.MethodPost, Path: web.ServiceInstancesURL, ResponseStatusCode: http.StatusAccepted, Headers: map[string]string{"Location": locationHeader}}
			})
			It("should receive operation location", func() {
				responseInstanceID, location, err := client.Provision(instance, serviceName, planName, params)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(location).Should(Equal(locationHeader))
				Expect(responseInstanceID).Should(Equal("12345"))
			})
		})

		Context("When invalid instance is being returned by SM", func() {
			BeforeEach(func() {
				responseBody, _ := json.Marshal(struct {
					Name bool `json:"name"`
				}{
					Name: true,
				})
				handlerDetails[0] = HandlerDetails{Method: http.MethodPost, Path: web.ServiceInstancesURL, ResponseBody: responseBody, ResponseStatusCode: http.StatusCreated}

			})
			It("should return error", func() {
				responseInstanceID, location, err := client.Provision(instance, serviceName, planName, params)

				Expect(err).Should(HaveOccurred())
				Expect(location).Should(BeEmpty())
				Expect(responseInstanceID).Should(BeEmpty())
			})
		})

		Context("When invalid status code is returned by SM", func() {
			Context("And status code is unsuccessful", func() {
				BeforeEach(func() {
					responseBody, _ := json.Marshal(instance)
					handlerDetails[0] = HandlerDetails{Method: http.MethodPost, Path: web.ServiceInstancesURL, ResponseBody: responseBody, ResponseStatusCode: http.StatusOK}
				})
				It("should return error with status code", func() {
					responseInstanceID, location, err := client.Provision(instance, serviceName, planName, params)

					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())
					verifyErrorMsg(err.Error(), handlerDetails[0].Path, handlerDetails[0].ResponseBody, handlerDetails[0].ResponseStatusCode)
					Expect(responseInstanceID).Should(BeEmpty())
				})
			})

			Context("And status code is unsuccessful", func() {
				BeforeEach(func() {
					responseBody := []byte(`{ "description": "description", "error": "error"}`)
					handlerDetails[0] = HandlerDetails{Method: http.MethodPost, Path: web.ServiceInstancesURL, ResponseBody: responseBody, ResponseStatusCode: http.StatusBadRequest}

				})
				It("should return error with url and description", func() {
					responseInstanceID, location, err := client.Provision(instance, serviceName, planName, params)

					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())
					verifyErrorMsg(err.Error(), handlerDetails[0].Path, handlerDetails[0].ResponseBody, handlerDetails[0].ResponseStatusCode)
					Expect(responseInstanceID).Should(BeEmpty())
				})
			})

			Context("And invalid response body", func() {
				BeforeEach(func() {
					responseBody := []byte(`{ "description": description", "error": "error"}`)
					handlerDetails[0] = HandlerDetails{Method: http.MethodPost, Path: web.ServiceInstancesURL, ResponseBody: responseBody, ResponseStatusCode: http.StatusBadRequest}

				})
				It("should return error without url and description if invalid response body", func() {
					responseInstanceID, location, err := client.Provision(instance, serviceName, planName, params)

					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())

					verifyErrorMsg(err.Error(), handlerDetails[0].Path, handlerDetails[0].ResponseBody, handlerDetails[0].ResponseStatusCode)
					Expect(responseInstanceID).Should(BeEmpty())
				})
			})

		})

	})

	Describe("Deprovision", func() {
		Context("when an existing instance is being deleted synchronously", func() {
			BeforeEach(func() {
				responseBody := []byte("{}")
				handlerDetails = []HandlerDetails{
					{Method: http.MethodDelete, Path: web.ServiceInstancesURL + "/", ResponseBody: responseBody, ResponseStatusCode: http.StatusOK},
				}
			})
			It("should be successfully removed", func() {
				location, err := client.Deprovision(instance.ID, params)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(location).Should(BeEmpty())
			})
		})

		Context("when an existing instance is being deleted asynchronously", func() {
			var locationHeader string
			BeforeEach(func() {
				locationHeader = "location"
				handlerDetails = []HandlerDetails{
					{Method: http.MethodDelete, Path: web.ServiceInstancesURL + "/", ResponseStatusCode: http.StatusAccepted, Headers: map[string]string{"Location": locationHeader}},
				}
			})
			It("should be successfully removed", func() {
				location, err := client.Deprovision(instance.ID, params)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(location).Should(Equal(locationHeader))
			})
		})

		Context("when service manager returns a non-expected status code", func() {
			BeforeEach(func() {
				responseBody := []byte("{}")

				handlerDetails = []HandlerDetails{
					{Method: http.MethodDelete, Path: web.ServiceInstancesURL + "/", ResponseBody: responseBody, ResponseStatusCode: http.StatusCreated},
				}
			})
			It("should handle error", func() {
				location, err := client.Deprovision(instance.ID, params)
				Expect(err).Should(HaveOccurred())
				Expect(location).Should(BeEmpty())
				verifyErrorMsg(err.Error(), handlerDetails[0].Path+instance.ID, handlerDetails[0].ResponseBody, handlerDetails[0].ResponseStatusCode)
			})
		})

		Context("when service manager returns a status code not found", func() {
			BeforeEach(func() {
				responseBody := []byte(`{ "description": "Instance not found" }`)

				handlerDetails = []HandlerDetails{
					{Method: http.MethodDelete, Path: web.ServiceInstancesURL + "/", ResponseBody: responseBody, ResponseStatusCode: http.StatusNotFound},
				}
			})
			It("should be considered as success", func() {
				location, err := client.Deprovision(instance.ID, params)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(location).Should(BeEmpty())
			})
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			offering := &types.ServiceOffering{
				ID:          "service_id",
				Name:        serviceName,
				CatalogName: serviceName,
			}
			plan := &types.ServicePlan{
				ID:                planID,
				Name:              planName,
				CatalogName:       serviceName,
				ServiceOfferingID: serviceID,
			}

			plansArray := []types.ServicePlan{*plan}
			plans := types.ServicePlans{ServicePlans: plansArray}
			plansBody, _ := json.Marshal(plans)
			offeringArray := []types.ServiceOffering{*offering}
			offerings := types.ServiceOfferings{ServiceOfferings: offeringArray}
			offeringResponseBody, _ := json.Marshal(offerings)
			handlerDetails = []HandlerDetails{
				{Method: http.MethodGet, Path: web.ServiceOfferingsURL, ResponseBody: offeringResponseBody, ResponseStatusCode: http.StatusOK},
				{Method: http.MethodGet, Path: web.ServicePlansURL, ResponseBody: plansBody, ResponseStatusCode: http.StatusOK},
			}
		})
		Context("When valid instance is being updated synchronously", func() {
			BeforeEach(func() {
				responseBody, _ := json.Marshal(instance)
				handlerDetails = append(handlerDetails,
					HandlerDetails{Method: http.MethodPatch, Path: web.ServiceInstancesURL + "/" + instance.ID, ResponseBody: responseBody, ResponseStatusCode: http.StatusOK})
			})
			It("should update successfully", func() {
				responseInstance, location, err := client.UpdateInstance(instance.ID, instance, serviceName, planName, params)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(location).Should(HaveLen(0))
				Expect(responseInstance).To(Equal(instance))
			})
		})

		Context("When valid instance is being updated asynchronously", func() {
			var locationHeader string
			BeforeEach(func() {
				locationHeader = "test-location"
				handlerDetails = append(handlerDetails,
					HandlerDetails{Method: http.MethodPatch, Path: web.ServiceInstancesURL + "/" + instance.ID, ResponseStatusCode: http.StatusAccepted, Headers: map[string]string{"Location": locationHeader}})
			})
			It("should receive operation location", func() {
				responseInstance, location, err := client.UpdateInstance(instance.ID, instance, serviceName, planName, params)

				Expect(err).ShouldNot(HaveOccurred())
				Expect(location).Should(Equal(locationHeader))
				Expect(responseInstance).To(BeNil())
			})
		})

		Context("When invalid instance is being returned by SM", func() {
			BeforeEach(func() {
				responseBody, _ := json.Marshal(struct {
					Name bool `json:"name"`
				}{
					Name: true,
				})
				handlerDetails = append(handlerDetails,
					HandlerDetails{Method: http.MethodPatch, Path: web.ServiceInstancesURL + "/" + instance.ID, ResponseBody: responseBody, ResponseStatusCode: http.StatusOK})
			})
			It("should return error", func() {
				responseInstance, location, err := client.UpdateInstance(instance.ID, instance, serviceName, planName, params)

				Expect(err).Should(HaveOccurred())
				Expect(location).Should(BeEmpty())
				Expect(responseInstance).To(BeNil())
			})
		})

		Context("When invalid status code is returned by SM", func() {
			Context("And status code is unsuccessful", func() {
				BeforeEach(func() {
					responseBody, _ := json.Marshal(instance)
					handlerDetails = append(handlerDetails,
						HandlerDetails{Method: http.MethodPatch, Path: web.ServiceInstancesURL + "/" + instance.ID, ResponseBody: responseBody, ResponseStatusCode: http.StatusTeapot})
				})
				It("should return error with status code", func() {
					responseInstance, location, err := client.UpdateInstance(instance.ID, instance, serviceName, planName, params)

					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())
					verifyErrorMsg(err.Error(), handlerDetails[2].Path, handlerDetails[2].ResponseBody, handlerDetails[2].ResponseStatusCode)
					Expect(responseInstance).To(BeNil())
				})
			})

			Context("And status code is unsuccessful", func() {
				BeforeEach(func() {
					responseBody := []byte(`{ "description": "description", "error": "error"}`)
					handlerDetails = append(handlerDetails,
						HandlerDetails{Method: http.MethodPatch, Path: web.ServiceInstancesURL + "/" + instance.ID, ResponseBody: responseBody, ResponseStatusCode: http.StatusBadRequest})
				})
				It("should return error with url and description", func() {
					responseInstance, location, err := client.UpdateInstance(instance.ID, instance, serviceName, planName, params)

					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())
					verifyErrorMsg(err.Error(), handlerDetails[2].Path, handlerDetails[2].ResponseBody, handlerDetails[2].ResponseStatusCode)
					Expect(responseInstance).To(BeNil())
				})
			})

			Context("And invalid response body", func() {
				BeforeEach(func() {
					responseBody := []byte(`{ "description": description", "error": "error"}`)
					handlerDetails = append(handlerDetails,
						HandlerDetails{Method: http.MethodPatch, Path: web.ServiceInstancesURL + "/" + instance.ID, ResponseBody: responseBody, ResponseStatusCode: http.StatusBadRequest})
				})
				It("should return error without url and description if invalid response body", func() {
					responseInstance, location, err := client.UpdateInstance(instance.ID, instance, serviceName, planName, params)

					Expect(err).Should(HaveOccurred())
					Expect(location).Should(BeEmpty())

					verifyErrorMsg(err.Error(), handlerDetails[2].Path, handlerDetails[2].ResponseBody, handlerDetails[2].ResponseStatusCode)
					Expect(responseInstance).To(BeNil())
				})
			})

		})

		Context("When invalid config is set", func() {
			It("should return error", func() {
				client, _ = smclient.NewClient(context.TODO(), &smclient.ClientConfig{URL: "invalidURL"}, fakeAuthClient)
				_, location, err := client.UpdateInstance(instance.ID, instance, serviceName, planName, params)

				Expect(err).Should(HaveOccurred())
				Expect(location).Should(BeEmpty())
			})
		})
	})
})

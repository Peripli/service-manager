package context_signature

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

var (
	ctx                   *common.TestContext
	brokerServer          *common.BrokerServer
	osbURL                string
	brokerID              string
	catalogServiceID      string
	catalogPlanID         string
	serviceID             string
	planID                string
	privateKeyStr         string
	publicKeyStr          string
	publicSuccessorKeyStr string
)

func TestContextSignature(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Context Signature Tests Suite")
}

var _ = BeforeSuite(func() {
	//private key was generated by:
	//	1. openssl genrsa -out rsa_key.pem 4096
	//  2. cat rsa_key.pem | base64
	privateKeyStr = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlKSndJQkFBS0NBZ0VBb0FSNkJBNWY3bG9ncFpTUXM2bHk3WFg0N0RDWDVIRVZMUklqdU4xYng4WUdPbG9jCm9mNS9uNHJBL2RBUjRuVzhlMXlhVFRyLzMwV3hONUtzSDFPaW5FMkovcGF5TWVuUmlFdTZMSFZIVUtlN3BLci8KRFBIR3g1dndoeXlqbnE5RXpycEdjVmg2VFNxVVBieEZTOHM0R2kzdUJXQUxEcmVOVUhQamJJMWE5RGh6L1lsZwp3RWloM2o2OVhlZjhDdStZQktpMkpDVVQwTXZMTVdrMG1NWm9NbEZyaDJ0SkFOd25YSUd1V0VNRmcxMHNpOFZrCmNNQ0hDNldVSkxUcGc5QWVSL0szS2JjdGdXV1RPMi9PZkhYazhGT3Y2OXRDOFBIQ3JyWUlEeW91Z1g0TlF0aC8KNjVZNzRmOUZjSjZNNHpJQzkrTWZtaFpJZjBZWDFZYlpKL2h6VXpFUGd3TmNzUEJTYVEzZG8ydkh1ZEQ4TVhNaQpiYWJXZTdscXdRbGNWSXhTQlF5UXVSdlA2SHhTaHozYzdGbjYwTWF4NjNydkxldjNwTlNROVh3QjhzR2dmSmR4CndpS0M2TUkramF4bDZLak5zM01yUE1NMXNSc1VEaHVBMWFSeW1OSTNKWkpLenlQTnBVcE03UXhSeHdYdlNVQWUKZ1g1aVBaRlpyOVI1bVlzTzJyb1dhWDljZmpZT1ErbGZqN2JFSUIwYy9MOXRYdHVrTmNEdDZ3TXVLcTFwRThpawo3bExDT2ZGQ3hOTzlQeE9iYkxPanNidjdZZGZMM0hkSTRhRnlkR3JNeS83WkM4eHNWdUxPOHZLVzFocUFCNzBUCmpYWXBobXpYU2U3MEhtUUJkY3BLN2NKaTh2T3ZHQktzb2t3NzVyYnNJMWllWlZvbGFxSzVSU3Z1T29NQ0F3RUEKQVFLQ0FnQVovNkF6ZUlKdG40Y2VYLzBDczgxUWQ1SnlEWk1nTXA5V0sxUlNmT1NrbUsvNld4bTcyRFcwSGo4cwovZGxxQ2VjTnhBWHQ5bUFNVHE1MGNRZzJMc2lFekxSWEFQVUMxeEtNS29HZEo1RG1zZG55N3pWeFRQY1hCMmNWCkQxT21QS1BaVXJxUFAramZFTVAxSTltK2JzNDJzcSt4ZitGTUN0YVM4OEZIcWMvVlRqYktRci9OZmYrT3RITGcKQndrVVhjazlPSXdmWTBiTTdjK2R2NUlrSUZoZGxJejcrNXBvNFZ3ajA0NFlHUXVVUkZjd2ZtbkxSL3lwRDhYNgpFTXEvOTloenFDUEtTMURCYlZkMm1MdmJ6T3ZkZ0R0Yy9zcnBpdDR1dExTcWdoZjhRaGExZmFlTEIyWERXazVWCjlleStIU29PVElDZDhIRG0vT1J5ZE81amFDS1VaK01vRzRzVGR5OU5zS3FOdjFwaXNSVjNhM0V5QzBZbW9BK1oKL1VaTTJ2bndENk1NaXR4UW1OOWczWndsOEFyTzVXYTZnRGNLRnRQa2NoanZTMXhaYXVsR2FUUjltQldMdEF1bQpTUW55aUprRHFmQWVGZTVjcWxIQ3NNMWRRUlZCNjk3MFpzUEc3ZVZ6NXVJa214QzRVS3hjYSt0QmI4ejRkaStBCmxMSGxrV1RMQ2tEUTIxeHU2WWFORU50WnV1MjhxVEw2Y3FCNnNQQkoyZGNDcGc5QUhDd1c2ck5kNk9zSnRNYTIKSXdSdEpQSVFIa1NuNzBrMFgvT0g4R0szZzgyaVB6OWJoN1RGdklWN3RsREt5bllOSG1zZi9RbHBvYy8waFo3Ywo0RGpDTnRkQ241NGY5K1NSMUg1Q3VqLzVqSC9YaHVnSGY3NDhza1A3dWZXSnQ4U09RUUtDQVFFQXp2TUEzd1JPClplbFg3UVVxODV6NU1Bd2JTNSs3amlNN2E0WFlvN05pUmYxRHBObW5iT1VaUGZOVWlXSEhleUVMcE9KaklWSlEKM1pKak56YnNBQy9hZ0pMVFRxNGVyZURibkZST2tGdDV6OUhjTXlWL3dMT3dlVjZFcFF6STFFRkJXMTRSb3Z3Lwo2eU8wU0Jxa2Q1NGRZKy9GaDloM0RDbVhyTHdYVEU5M1NuTkFUeE5PWGU2Y00vbnRhdWZMVlkrOXQ3aDJzM0pLCmk0cUk0ZGFCNzhpaHRaQlBNOXVocWxsd1dkZ3loS0FCQ3kvKzVDMWRNN0pvL3FZK1ZIR0RKdVk1bW0yZVJhZkcKMFdUNjJ2UGdIWUlMQzFuSlV1eitrTjduNHlDTVdqbFk3TlowNURVQ0RkRTNnZUNVOEE5dE12TGt0UWc4UHdGNQo5VGgrSFB5SDM0NTR4UUtDQVFFQXhmSFBUTm9MS2w0dHlZVFhvVkxtMGZyWVpIZG9oRFl2em1ycXg3NmhFL2ZJCnJSTG56ZHpnMEZzZUFVdU5OV0phdzlDd1lhZGkrVWw3Sysra05odHowdmpiYktsUk44ZDNSNWJvb3o5dW1CUzkKKzZicWNybTN1RUFYV0hLTzNyLzdzVXdXcnExdjFJc2YxWjhQNkovMmlMQ0l0cjRmVmw3cXJIUkY2eWlLYTNFdQp1NElxYXNaTmtzY05hNXZkTzRUN3RzSjlFSWJIdXBWSDEvUk1HbmVvbnhRZ0tabVYxY2NwWVNyV2pBWDNPRlNjCmlreHhyRGwvUTRMblVub1dsSlR1aUFQQWY2MHlHYm1tWDZVeFZlMHVsUUtCNjhCMW56NG9UeW1CWnRGb2s1a3QKV3dJZXVPSkhvZU4rNUR0b3V0Q0NJamFsVTQrS293aXJMV3lqbHdyS3B3S0NBUUEzNHNPZmRpZzl1Uy8zWCtmagpkY2FOUlJleDZtYlowWVhnV1hyUmFrWGxwS2s1d1ZWSFFPNzZIZFg4YTUxVkVPMTJEM1M1c09NSmt0aWNOb2F3CjNqdGhjVVVEQUY1a2trNTcvd0JnVjNPanZZWjdnV3JvZlIzeENLZEZjeGhneVdaKzUvNVhSMHR6a21iQytmN2sKRnB6Vk9oRGJ5SWNOajhYWDdjdFUzampXc0J6enZjRHgrTmZSNlhKRjVtYXdxbXFQVEk4eGtuR3pFU3c1NEpXaQpUVW51SUJSamFySlRzR2Q0dTd1WXVTVFBBcDBRdlhkbjJJd21DSHJZanZiZDhGb3A2K1JMNXl6M3F3OWJSWFNHClEzSDhGTGtiWGNpNUVwa0lhdWU4RGJTSDhMb01Ub3hKY3ZCTWNIdUlBSUo2dWNFdGFoWHE1ZGtyY2FBTTc5MDUKYjk0SkFvSUJBQVVaTFZXMWFBUTNXTWFQL1YzU1hNK2J2bWNZREVFYmhDKzA0VWN6eWNKUjU0Rk5zMXJDRGFoUQpNSDJvRElNTGZYcjlyUTFXMmwzQlhzTEs4VmZUYlRCSjZKenIzNE9vUjVJNGVOVjdsTVdtQXg2d05lbXVqdVRZCkFjSHRjWENiVVVoSHhXM0tXYzhIcGxKQ1BvNm5VQnBGTWNCRE5WdHNKbTg2cjNKWElQbVRlTGlycVp3R2I0a1EKUjNBMkc0U2s4RGJNMjV2SlhPdVpYTGhiT25xVUNtdk9nT1dSWnlLU2RxWmlEQlNmTXJib3R2OTQ2SlNmQm9BZQpwd2Fnem1RVlVlOSs2VDVnbjZHNS9tY0lRalVNWHQ3SHFjRUF2QWJWK3dQTzlkNUlGb0YydUl4WGlhTUpjUDdpCmRTbzd2WDdTVUFmQmtKQ09hZXU1RlcrZUZMaVhOcEVDZ2dFQVRzOThxYzNCSlh1ZVRnK3dtN0JXbGM2UWU4OFAKUlhES3o0UklELzZYOE93QzdiWDY2OHl6aU02aUlML1JDUTZ5NjFHYnFZSVVjQ0h3N3pkczloVTdMN2JCV1BDeApMd0kvNE4wUHZ5UkFwMzJITG45RmxnT3IrUDNWUWdrWjVWWXlVZGRDU2lrV0V0OWMyNk5sczJWSFZaVVRLVnJqCmE1cTVKdVdDTFUzTFVDNUdvVGgwVGFaR0w4N2JzdjBscEg1MnA4OFB5T09hb3hBaGY3MUkzMWd3U3drNjdXRHQKcld1dnFwbG9kVU4wWWVSbTNpdGR1SU9USkRMbmpHemRsT3VSb2piRUJRZkNMSXpvZUpPc215cnR1akxBQ2xHQgppWU40SWtqNnZhNVBlM0duc1dUTnFlTU95YzM2eW03bjJRU291aXdtVE00QkVKc25mWmVtSkRCeUlnPT0KLS0tLS1FTkQgUlNBIFBSSVZBVEUgS0VZLS0tLS0K"
	//public key was generated by:
	//	1. openssl rsa -in rsa_key.pem -pubout -out rsa_pub_key.pem
	//	2. cat rsa_pub_key.pem | base64
	publicKeyStr = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQ0lqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FnOEFNSUlDQ2dLQ0FnRUFvQVI2QkE1Zjdsb2dwWlNRczZseQo3WFg0N0RDWDVIRVZMUklqdU4xYng4WUdPbG9jb2Y1L240ckEvZEFSNG5XOGUxeWFUVHIvMzBXeE41S3NIMU9pCm5FMkovcGF5TWVuUmlFdTZMSFZIVUtlN3BLci9EUEhHeDV2d2h5eWpucTlFenJwR2NWaDZUU3FVUGJ4RlM4czQKR2kzdUJXQUxEcmVOVUhQamJJMWE5RGh6L1lsZ3dFaWgzajY5WGVmOEN1K1lCS2kySkNVVDBNdkxNV2swbU1abwpNbEZyaDJ0SkFOd25YSUd1V0VNRmcxMHNpOFZrY01DSEM2V1VKTFRwZzlBZVIvSzNLYmN0Z1dXVE8yL09mSFhrCjhGT3Y2OXRDOFBIQ3JyWUlEeW91Z1g0TlF0aC82NVk3NGY5RmNKNk00eklDOStNZm1oWklmMFlYMVliWkovaHoKVXpFUGd3TmNzUEJTYVEzZG8ydkh1ZEQ4TVhNaWJhYldlN2xxd1FsY1ZJeFNCUXlRdVJ2UDZIeFNoejNjN0ZuNgowTWF4NjNydkxldjNwTlNROVh3QjhzR2dmSmR4d2lLQzZNSStqYXhsNktqTnMzTXJQTU0xc1JzVURodUExYVJ5Cm1OSTNKWkpLenlQTnBVcE03UXhSeHdYdlNVQWVnWDVpUFpGWnI5UjVtWXNPMnJvV2FYOWNmallPUStsZmo3YkUKSUIwYy9MOXRYdHVrTmNEdDZ3TXVLcTFwRThpazdsTENPZkZDeE5POVB4T2JiTE9qc2J2N1lkZkwzSGRJNGFGeQpkR3JNeS83WkM4eHNWdUxPOHZLVzFocUFCNzBUalhZcGhtelhTZTcwSG1RQmRjcEs3Y0ppOHZPdkdCS3Nva3c3CjVyYnNJMWllWlZvbGFxSzVSU3Z1T29NQ0F3RUFBUT09Ci0tLS0tRU5EIFBVQkxJQyBLRVktLS0tLQo="

	publicSuccessorKeyStr = "testKey"

	ctx = common.NewTestContextBuilderWithSecurity().WithEnvPostExtensions(func(e env.Environment, servers map[string]common.FakeServer) {
		e.Set("api.osb_rsa_private_key", privateKeyStr)
		e.Set("api.osb_rsa_public_key", publicKeyStr)
		e.Set("api.osb_successor_rsa_public_key", publicSuccessorKeyStr)
	}).Build()
	UUID, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	catalogPlanID = UUID.String()
	plan1 := common.GenerateTestPlanWithID(catalogPlanID)
	UUID, err = uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	catalogServiceID = UUID.String()
	service1 := common.GenerateTestServiceWithPlansWithID(catalogServiceID, plan1)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(service1)

	brokerID, _, brokerServer = ctx.RegisterBrokerWithCatalog(catalog).GetBrokerAsParams()
	brokerServer.ShouldRecordRequests(true)
	common.CreateVisibilitiesForAllBrokerPlans(ctx.SMWithOAuth, brokerID)
	osbURL = "/v1/osb/" + brokerID

	serviceOfferings := ctx.SMWithBasic.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerID)).First()
	serviceID = serviceOfferings.Object().Value("id").String().Raw()
	servicePlans := ctx.SMWithBasic.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", serviceID))
	planID = servicePlans.Element(0).Object().Value("id").String().Raw()
})

var _ = AfterSuite(func() {
	ctx.Cleanup()
})

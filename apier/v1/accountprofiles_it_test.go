// +build integration

/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package v1

import (
	"net/rpc"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/ericlagergren/decimal"

	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
)

var (
	accPrfCfgPath   string
	accPrfCfg       *config.CGRConfig
	accSRPC         *rpc.Client
	accPrfDataDir   = "/usr/share/cgrates"
	apiAccPrf       *APIAccountProfileWithCache
	accPrf          *utils.AccountProfile
	accPrfConfigDIR string //run tests for specific configuration

	sTestsAccPrf = []func(t *testing.T){
		testAccountSInitCfg,
		testAccountSInitDataDb,
		testAccountSResetStorDb,
		testAccountSStartEngine,
		testAccountSRPCConn,
		testAccountSLoadFromFolder,
		testAccountSGetAccountProfile,
		testAccountSPing,
		testAccountSSettAccountProfile,
		testAccountSGetAccountProfileIDs,
		testAccountSGetAccountProfileIDsCount,
		testAccountSUpdateAccountProfile,
		testAccountSRemoveAccountProfile,
		testAccountSKillEngine,
	}
)

//Test start here
func TestAccountSIT(t *testing.T) {
	switch *dbType {
	case utils.MetaInternal:
		accPrfConfigDIR = "tutinternal"
	case utils.MetaMySQL:
		accPrfConfigDIR = "tutmysql"
	case utils.MetaMongo:
		accPrfConfigDIR = "tutmongo"
	case utils.MetaPostgres:
		t.SkipNow()
	default:
		t.Fatal("Unknown Database type")
	}
	for _, stest := range sTestsAccPrf {
		t.Run(accPrfConfigDIR, stest)
	}
}

func testAccountSInitCfg(t *testing.T) {
	var err error
	accPrfCfgPath = path.Join(accPrfDataDir, "conf", "samples", accPrfConfigDIR)
	accPrfCfg, err = config.NewCGRConfigFromPath(accPrfCfgPath)
	if err != nil {
		t.Error(err)
	}
}

func testAccountSInitDataDb(t *testing.T) {
	if err := engine.InitDataDb(accPrfCfg); err != nil {
		t.Fatal(err)
	}
}

// Wipe out the cdr database
func testAccountSResetStorDb(t *testing.T) {
	if err := engine.InitStorDb(accPrfCfg); err != nil {
		t.Fatal(err)
	}
}

// Start CGR Engine
func testAccountSStartEngine(t *testing.T) {
	if _, err := engine.StopStartEngine(accPrfCfgPath, *waitRater); err != nil {
		t.Fatal(err)
	}
}

// Connect rpc client to rater
func testAccountSRPCConn(t *testing.T) {
	var err error
	accSRPC, err = newRPCClient(accPrfCfg.ListenCfg()) // We connect over JSON so we can also troubleshoot if needed
	if err != nil {
		t.Fatal(err)
	}
}

func testAccountSLoadFromFolder(t *testing.T) {
	var reply string
	acts := &utils.AttrLoadTpFromFolder{FolderPath: path.Join(*dataDir, "tariffplans", "tutaccounts")}
	if err := accSRPC.Call(utils.APIerSv1LoadTariffPlanFromFolder, acts, &reply); err != nil {
		t.Error(err)
	}
	time.Sleep(100 * time.Millisecond)
}

func testAccountSGetAccountProfile(t *testing.T) {
	expected := &utils.AccountProfile{
		Tenant:    "cgrates.org",
		ID:        "1001",
		FilterIDs: []string{},
		Weight:    20,
		Balances: map[string]*utils.Balance{
			"MonetaryBalance": &utils.Balance{
				ID:        "MonetaryBalance",
				FilterIDs: []string{},
				Weight:    10,
				Type:      utils.MetaMonetary,
				CostIncrements: []*utils.CostIncrement{
					&utils.CostIncrement{
						FilterIDs:    []string{"fltr1", "fltr2"},
						Increment:    &utils.Decimal{decimal.New(13, 1)},
						FixedFee:     &utils.Decimal{decimal.New(23, 1)},
						RecurrentFee: &utils.Decimal{decimal.New(33, 1)},
					},
				},
				AttributeIDs:   []string{"attr1", "attr2"},
				RateProfileIDs: []string{},
				UnitFactors: []*utils.UnitFactor{
					&utils.UnitFactor{
						FilterIDs: []string{"fltr1", "fltr2"},
						Factor:    &utils.Decimal{decimal.New(100, 0)},
					},
					&utils.UnitFactor{
						FilterIDs: []string{"fltr3"},
						Factor:    &utils.Decimal{decimal.New(200, 0)},
					},
				},
				Units: &utils.Decimal{decimal.New(14, 0)},
			},
			"VoiceBalance": &utils.Balance{
				ID:             "VoiceBalance",
				FilterIDs:      []string{},
				Weight:         10,
				Type:           utils.MetaVoice,
				CostIncrements: []*utils.CostIncrement{},
				AttributeIDs:   []string{},
				RateProfileIDs: []string{},
				UnitFactors:    []*utils.UnitFactor{},
				Units:          &utils.Decimal{decimal.New(3600000000000, 0)},
			},
		},
		ThresholdIDs: []string{utils.MetaNone},
	}
	if *encoding == utils.MetaGOB {
		expected.FilterIDs = nil
		for i := range expected.Balances {
			expected.Balances[i].FilterIDs = nil
		}
	}
	var reply *utils.AccountProfile
	if err := accSRPC.Call(utils.APIerSv1GetAccountProfile,
		utils.TenantIDWithOpts{TenantID: &utils.TenantID{Tenant: "cgrates.org", ID: "1001"}}, &reply); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(expected, reply) {
		t.Errorf("Expecting : %+v \n received: %+v", utils.ToJSON(expected), utils.ToJSON(reply))
	}
}

func testAccountSPing(t *testing.T) {
	var resp string
	if err := accSRPC.Call(utils.AccountSv1Ping, new(utils.CGREvent), &resp); err != nil {
		t.Error(err)
	} else if resp != utils.Pong {
		t.Error("Unexpected reply returned", resp)
	}
}

func testAccountSSettAccountProfile(t *testing.T) {
	apiAccPrf = &APIAccountProfileWithCache{
		APIAccountProfileWithOpts: &utils.APIAccountProfileWithOpts{
			APIAccountProfile: &utils.APIAccountProfile{
				Tenant: "cgrates.org",
				ID:     "id_test",
				Weight: 10,
				Balances: map[string]*utils.APIBalance{
					"MonetaryBalance": &utils.APIBalance{
						ID:        "MonetaryBalance",
						FilterIDs: []string{},
						Weight:    10,
						Type:      utils.MetaMonetary,
						CostIncrements: []*utils.APICostIncrement{
							&utils.APICostIncrement{
								FilterIDs:    []string{"fltr1", "fltr2"},
								Increment:    utils.Float64Pointer(1.3),
								FixedFee:     utils.Float64Pointer(2.3),
								RecurrentFee: utils.Float64Pointer(3.3),
							},
						},
						AttributeIDs: []string{"attr1", "attr2"},
						UnitFactors: []*utils.APIUnitFactor{
							&utils.APIUnitFactor{
								FilterIDs: []string{"fltr1", "fltr2"},
								Factor:    100,
							},
							&utils.APIUnitFactor{
								FilterIDs: []string{"fltr3"},
								Factor:    200,
							},
						},
						Units: 14,
					},
					"VoiceBalance": &utils.APIBalance{
						ID:        "VoiceBalance",
						FilterIDs: []string{},
						Weight:    10,
						Type:      utils.MetaVoice,
						Units:     3600000000000,
					},
				},
				ThresholdIDs: []string{utils.MetaNone},
			},
			Opts: map[string]interface{}{},
		},
	}
	var result string
	expErr := utils.ErrNotFound.Error()
	if err := accSRPC.Call(utils.APIerSv1GetAccountProfile, &utils.TenantIDWithOpts{
		TenantID: &utils.TenantID{Tenant: "cgrates.org", ID: "id_test"}}, &result); err == nil || err.Error() != expErr {
		t.Errorf("Expected error: %v received: %v", expErr, err)
	}
	var reply string
	if err := accSRPC.Call(utils.APIerSv1SetAccountProfile, apiAccPrf, &reply); err != nil {
		t.Error(err)
	} else if reply != utils.OK {
		t.Error("Unexpected reply returned", reply)
	}
	var err error
	if accPrf, err = apiAccPrf.AsAccountProfile(); err != nil {
		t.Error(err)
	}
	var reply2 *utils.AccountProfile
	if err := accSRPC.Call(utils.APIerSv1GetAccountProfile, &utils.TenantIDWithOpts{
		TenantID: &utils.TenantID{Tenant: "cgrates.org", ID: "id_test"}}, &reply2); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(accPrf, reply2) {
		t.Errorf("Expecting : %+v, received: %+v", accPrf, reply2)
	}

}

func testAccountSGetAccountProfileIDs(t *testing.T) {
	expected := []string{"id_test", "1001"}
	var result []string
	if err := accSRPC.Call(utils.APIerSv1GetAccountProfileIDs, utils.PaginatorWithTenant{}, &result); err != nil {
		t.Error(err)
	} else if len(expected) != len(result) {
		t.Errorf("Expecting : %+v, received: %+v", expected, result)
	}
	if err := accSRPC.Call(utils.APIerSv1GetAccountProfileIDs, utils.PaginatorWithTenant{Tenant: "cgrates.org"}, &result); err != nil {
		t.Error(err)
	} else if len(expected) != len(result) {
		t.Errorf("Expecting : %+v, received: %+v", expected, result)
	}
	if err := accSRPC.Call(utils.APIerSv1GetAccountProfileIDs, utils.PaginatorWithTenant{
		Tenant:    "cgrates.org",
		Paginator: utils.Paginator{Limit: utils.IntPointer(1)},
	}, &result); err != nil {
		t.Error(err)
	} else if 1 != len(result) {
		t.Errorf("Expecting : %+v, received: %+v", 1, result)
	}

}

func testAccountSGetAccountProfileIDsCount(t *testing.T) {
	var reply int
	if err := accSRPC.Call(utils.APIerSv1GetAccountProfileIDsCount,
		&utils.TenantWithOpts{Tenant: "cgrates.org"}, &reply); err != nil {
		t.Error(err)
	} else if reply != 2 {
		t.Errorf("Expecting: 2, received: %+v", reply)
	}

}

func testAccountSUpdateAccountProfile(t *testing.T) {
	var reply string
	apiAccPrf.Weight = 2
	apiAccPrf.Balances["MonetaryBalance"].CostIncrements[0].FixedFee = utils.Float64Pointer(123.5)
	if err := accSRPC.Call(utils.APIerSv1SetAccountProfile, apiAccPrf, &reply); err != nil {
		t.Error(err)
	} else if reply != utils.OK {
		t.Error("Unexpected reply returned", reply)
	}
	var err error
	if accPrf, err = apiAccPrf.AsAccountProfile(); err != nil {
		t.Error(err)
	}
	var reply2 *utils.AccountProfile
	if err := accSRPC.Call(utils.APIerSv1GetAccountProfile, &utils.TenantIDWithOpts{
		TenantID: &utils.TenantID{Tenant: "cgrates.org", ID: "id_test"}}, &reply2); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(accPrf, reply2) {
		t.Errorf("Expecting : %+v, received: %+v", accPrf, reply2)
	}
}

func testAccountSRemoveAccountProfile(t *testing.T) {
	var reply string
	if err := accSRPC.Call(utils.APIerSv1RemoveAccountProfile, &utils.TenantIDWithCache{Tenant: "cgrates.org", ID: "id_test"}, &reply); err != nil {
		t.Error(err)
	} else if reply != utils.OK {
		t.Error("Unexpected reply returned", reply)
	}
	var reply2 *utils.AccountProfile
	expErr := utils.ErrNotFound.Error()
	if err := accSRPC.Call(utils.APIerSv1GetAccountProfile, &utils.TenantIDWithOpts{
		TenantID: &utils.TenantID{Tenant: "cgrates.org", ID: "id_test"}}, &reply2); err == nil || err.Error() != expErr {
		t.Errorf("Expected error: %v received: %v", expErr, err)
	}
	if err := accSRPC.Call(utils.APIerSv1RemoveAccountProfile, &utils.TenantIDWithCache{Tenant: "cgrates.org", ID: "id_test"}, &reply2); err == nil || err.Error() != expErr {
		t.Errorf("Expected error: %v received: %v", expErr, err)
	}
}

func testAccountSKillEngine(t *testing.T) {
	if err := engine.KillEngine(100); err != nil {
		t.Error(err)
	}
}

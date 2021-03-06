/*
Real-time Online/Offline Charging System (OerS) for Telecom & ISP environments
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

package accounts

import (
	"fmt"
	"time"

	"github.com/cgrates/cgrates/engine"
	"github.com/cgrates/cgrates/utils"
	"github.com/ericlagergren/decimal"
)

// newAccountBalances constructs accountBalances
func newAccountBalances(acnt *utils.AccountProfile,
	fltrS *engine.FilterS, connMgr *engine.ConnManager,
	attrSConns, rateSConns []string) (acntBlncs *accountBalances, err error) {
	blncs := make(utils.Balances, len(acnt.Balances)) // Temporary code to pass the build please update this accordingly
	for _, bal := range acnt.Balances {
		blncs = append(blncs, bal)
	}
	blncs.Sort()
	acntBlncs = &accountBalances{blnCfgs: blncs, connMgr: connMgr, attrSConns: attrSConns, rateSConns: rateSConns}
	// populate typIdx
	for i, blnCfg := range blncs {
		acntBlncs.typIdx[blnCfg.Type] = append(acntBlncs.typIdx[blnCfg.Type], i)
	}
	// populate cncrtBlncs
	acntBlncs.cncrtBlncs = make([]*concreteBalance, len(acntBlncs.typIdx[utils.MetaConcrete]))
	for i, blncIdx := range acntBlncs.typIdx[utils.MetaConcrete] {
		acntBlncs.cncrtBlncs[i] = newConcreteBalanceOperator(acntBlncs.blnCfgs[blncIdx],
			acntBlncs.cncrtBlncs, fltrS, connMgr, attrSConns, rateSConns).(*concreteBalance)
		acntBlncs.opers[acntBlncs.blnCfgs[blncIdx].ID] = acntBlncs.cncrtBlncs[i]
	}
	// populate opers
	for _, blnCfg := range acntBlncs.blnCfgs {
		if blnCfg.Type == utils.MetaConcrete { // already computed above
			continue
		}
		if acntBlncs.opers[blnCfg.ID], err = newBalanceOperator(blnCfg,
			acntBlncs.cncrtBlncs, fltrS, connMgr, attrSConns, rateSConns); err != nil {
			return
		}
	}
	return
}

// accountBalances implements processing of the events centralized
type accountBalances struct {
	blnCfgs    []*utils.Balance           // ordered list of balance configurations
	typIdx     map[string][]int           // index based on type
	cncrtBlncs []*concreteBalance         // concrete balances so we can pass them to the newBalanceOperator
	opers      map[string]balanceOperator // map[blncID]balanceOperator

	fltrS   *engine.FilterS
	connMgr *engine.ConnManager
	attrSConns,
	rateSConns []string
}

// newBalanceOperator instantiates balanceOperator interface
// cncrtBlncs are needed for abstract balance debits
func newBalanceOperator(blncCfg *utils.Balance, cncrtBlncs []*concreteBalance,
	fltrS *engine.FilterS, connMgr *engine.ConnManager,
	attrSConns, rateSConns []string) (bP balanceOperator, err error) {
	switch blncCfg.Type {
	default:
		return nil, fmt.Errorf("unsupported balance type: <%s>", blncCfg.Type)
	case utils.MetaConcrete:
		return newConcreteBalanceOperator(blncCfg, cncrtBlncs, fltrS, connMgr, attrSConns, rateSConns), nil
	case utils.MetaAbstract:
		return newAbstractBalanceOperator(blncCfg, cncrtBlncs, fltrS, connMgr, attrSConns, rateSConns), nil
	}
}

// balanceOperator is the implementation of a balance type
type balanceOperator interface {
	debitUsage(usage *utils.Decimal, startTime time.Time,
		cgrEv *utils.CGREvent) (ec *utils.EventCharges, err error)
}

// roundUsageWithIncrements rounds the usage based on increments
func roundedUsageWithIncrements(usage, incrm *decimal.Big) (rndedUsage *decimal.Big) {
	usgMaxIncrm := decimal.WithContext(
		decimal.Context{RoundingMode: decimal.ToZero}).Quo(usage,
		incrm).RoundToInt()
	rndedUsage = utils.MultiplyBig(usgMaxIncrm, incrm)
	return
}

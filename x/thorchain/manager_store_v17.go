package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// Recently a failed node rescue cause the network to slash the node operator with all the yggdrasil funds , as a result of that there is a
// discrepancy between vault and pool , as you can see from the table below. This will credit those discrepancy back to pool , so LPs can
// benefit from it.
// +-----------------+------------------+---------------------+------------------+----------------------+------------------------+--------+
// |      ASSET      |   POOL BALANCE   | TOTAL VAULT BALANCE | TOTAL IN WALLET  | DIFF (POOL VS VAULT) | DIFF (VAULT VS WALLET) | STATUS |
// +-----------------+------------------+---------------------+------------------+----------------------+------------------------+--------+
// | BNB.AERGO-46B   |  121453.60723003 |     122702.67151353 |  122702.67151353 |         1249.0642835 |                      0 | FAIL   |
// | BNB.AVA-645     |  221906.47447619 |     225865.14267616 |  225865.14267616 |        3958.66819997 |                      0 | FAIL   |
// | BNB.AWC-986     |      35.32771717 |         35.32771717 |      35.32771717 |                    0 |                      0 | OK     |
// | BNB.BCH-1FD     |       24.7975236 |         24.98844201 |      24.98844201 |           0.19091841 |                      0 | FAIL   |
// | BNB.BEAR-14C    |    7767.62273314 |       7767.62273314 |    7767.62273314 |                    0 |                      0 | OK     |
// | BNB.BNB         |  132967.97679325 |     133585.16751505 |  133585.16760405 |          617.1907218 |               0.000089 | FAIL   |
// | BNB.BOLT-4C6    | 1104627.28505724 |    1118268.86957449 | 1118268.86957449 |       13641.58451725 |                      0 | FAIL   |
// | BNB.BTCB-1DE    |     106.15040783 |        106.59544551 |     106.59544551 |           0.44503768 |                      0 | FAIL   |
// | BNB.BULL-BE4    |       1.20093447 |          1.21132722 |       1.21132722 |           0.01039275 |                      0 | FAIL   |
// | BNB.BUSD-BD1    |  713859.18491123 |     716830.97207627 |  716830.97207627 |        2971.78716504 |                      0 | FAIL   |
// | BNB.BZNT-464    |   187623.3028437 |     188953.00094059 |  188953.00094059 |        1329.69809689 |                      0 | FAIL   |
// | BNB.CAN-677     |  871166.14723109 |     952569.63400502 |  952569.63400502 |       81403.48677393 |                      0 | FAIL   |
// | BNB.CAS-167     |               10 |                  10 |               10 |                    0 |                      0 | OK     |
// | BNB.CBIX-3C9    |   64063.61046991 |      64063.61046991 |   64063.61046991 |                    0 |                      0 | OK     |
// | BNB.COTI-CBB    |  394807.34553798 |     395740.45723139 |  395740.45723139 |         933.11169341 |                      0 | FAIL   |
// | BNB.CRPT-8C9    |                6 |                   6 |                6 |                    0 |                      0 | OK     |
// | BNB.DARC-24B    |   32894.52915902 |      32894.52915902 |   32894.52915902 |                    0 |                      0 | OK     |
// | BNB.DOS-120     |  145131.31149311 |     146296.23235962 |  146296.23235962 |        1164.92086651 |                      0 | FAIL   |
// | BNB.EOSBULL-F0D |     1739.2603444 |       1762.61042623 |    1762.61042623 |          23.35008183 |                      0 | FAIL   |
// | BNB.ERD-D06     |           29.746 |              29.746 |           29.746 |                    0 |                      0 | OK     |
// | BNB.ETH-1C9     |    1150.04276094 |       1153.87375588 |    1153.87375588 |           3.83099494 |                      0 | FAIL   |
// | BNB.ETHBEAR-B2B |             5010 |                5010 |             5010 |                    0 |                      0 | OK     |
// | BNB.ETHBULL-D33 |      10.64489692 |         10.71975435 |      10.71975435 |           0.07485743 |                      0 | FAIL   |
// | BNB.FRM-DE7     |  465451.53179903 |     469675.11088692 |  469675.11088692 |        4223.57908789 |                      0 | FAIL   |
// | BNB.FTM-A64     | 5105027.50186312 |    5125960.80304115 | 5125960.80304115 |       20933.30117803 |                      0 | FAIL   |
// | BNB.GIV-94E     |    3719.70526896 |       3719.70526896 |    3719.70526896 |                    0 |                      0 | OK     |
// | BNB.LINK-AAD    |           10.484 |              10.484 |           10.484 |                    0 |                      0 | OK     |
// | BNB.LIT-099     |     5478.9836171 |        5478.9836171 |     5478.9836171 |                    0 |                      0 | OK     |
// | BNB.LOKI-6A9    |    5527.55641922 |       5577.21830352 |    5577.21830352 |           49.6618843 |                      0 | FAIL   |
// | BNB.LTC-F07     |       5.96968308 |          5.96968308 |       5.96968308 |                    0 |                      0 | OK     |
// | BNB.LTO-BDF     |    2925.58407573 |       2925.58407573 |    2925.58407573 |                    0 |                      0 | OK     |
// | BNB.MITX-CAA    | 1312554.69233188 |    1329538.89658085 | 1329538.89658085 |       16984.20424897 |                      0 | FAIL   |
// | BNB.NEXO-A84    |     713.84392476 |        713.84392476 |     713.84392476 |                    0 |                      0 | OK     |
// | BNB.PROPEL-6D9  |             7500 |                7500 |             7500 |                    0 |                      0 | OK     |
// | BNB.QBX-38C     |    9777.61956919 |       9777.61956919 |    9777.61956919 |                    0 |                      0 | OK     |
// | BNB.RAVEN-F66   |  337753.85046373 |    1157463.26624621 | 1157463.26624621 |      819709.41578248 |                      0 | FAIL   |
// | BNB.SHR-DB6     |   44884.03304327 |      45478.93496549 |   45478.93496549 |         594.90192222 |                      0 | FAIL   |
// | BNB.SWINGBY-888 |  544942.78731416 |     549568.65951512 |  549568.65951512 |        4625.87220096 |                      0 | FAIL   |
// | BNB.TOMOB-4BC   |     471.03252552 |        471.03252552 |     471.03252552 |                    0 |                      0 | OK     |
// | BNB.TRXB-2E6    |             1720 |                1720 |             1720 |                    0 |                      0 | OK     |
// | BNB.TWT-8C2     | 1112618.35528414 |    1118550.29115396 | 1118550.29115396 |        5931.93586982 |                      0 | FAIL   |
// | BNB.UNI-DD8     |           9.9906 |              9.9906 |           9.9906 |                    0 |                      0 | OK     |
// | BNB.VIDT-F53    |      14.61708923 |         14.61708923 |      14.61708923 |                    0 |                      0 | OK     |
// | BNB.WISH-2D5    |    9033.35593128 |       9033.35593128 |    9033.35593128 |                    0 |                      0 | OK     |
// | BNB.XRP-BF2     |    2489.94988029 |       2522.01524947 |    2522.01524947 |          32.06536918 |                      0 | FAIL   |
// +-----------------+------------------+---------------------+------------------+----------------------+------------------------+--------+

var assetsToPool = map[string]int64{
	"BNB.AERGO-46B":   124906428350,
	"BNB.AVA-645":     395866819997,
	"BNB.BCH-1FD":     19091841,
	"BNB.BNB":         61719150131,
	"BNB.BOLT-4C6":    1364158451725,
	"BNB.BTCB-1DE":    44503768,
	"BNB.BULL-BE4":    1039275,
	"BNB.BUSD-BD1":    297178716504,
	"BNB.BZNT-464":    132969809689,
	"BNB.CAN-677":     8140348677393,
	"BNB.COTI-CBB":    93311169341,
	"BNB.DOS-120":     116492086651,
	"BNB.EOSBULL-F0D": 2335008183,
	"BNB.ETH-1C9":     383099494,
	"BNB.ETHBULL-D33": 7485743,
	"BNB.FRM-DE7":     422357908789,
	"BNB.FTM-A64":     2093330117803,
	"BNB.LOKI-6A9":    4966188430,
	"BNB.MITX-CAA":    1698420424897,
	"BNB.RAVEN-F66":   81970941578248,
	"BNB.SHR-DB6":     59490192222,
	"BNB.SWINGBY-888": 462587220096,
	"BNB.TWT-8C2":     593193586982,
	"BNB.XRP-BF2":     3206536918,
}

func (smgr *StoreMgr) calibrateVaultToPool(ctx cosmos.Context) error {
	network := common.GetCurrentChainNetwork()
	if network != common.MainNet {
		ctx.Logger().Info("not chaosnet , no need to update pool asset")
		return nil
	}
	for key, value := range assetsToPool {
		asset, err := common.NewAsset(key)
		if err != nil {
			ctx.Logger().Error("fail to parse asset", "asset", key)
			continue
		}
		p, err := smgr.keeper.GetPool(ctx, asset)
		if err != nil {
			ctx.Logger().Error("fail to get pool from key value store", "asset", asset.String())
			continue
		}

		if p.IsEmpty() {
			continue
		}
		oldBalance := p.BalanceAsset
		valueToPool := cosmos.NewUint(uint64(value))
		p.BalanceAsset = p.BalanceAsset.Add(valueToPool)
		if err := smgr.keeper.SetPool(ctx, p); err != nil {
			ctx.Logger().Error("fail to save pool back to key value store", "asset", asset.String())
		} else {
			ctx.Logger().Info(fmt.Sprintf("successfully added %s to %s pool , pool asset balance before change: %s, new pool asset balance is: %s", valueToPool, p.Asset.String(), oldBalance, p.BalanceAsset))
		}
	}
	return nil
}

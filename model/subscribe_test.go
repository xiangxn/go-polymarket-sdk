package model

import (
	"encoding/json"
	"testing"
)

const rawTradeJSON = `{
    "type": "TRADE",
    "id": "55beb29d-e2d6-4e49-a6ca-9e1316596de1",
    "taker_order_id": "0x0b416f20bfbd51176520950c16a6188f306c1b12abe47b531c6ba7468e2dfef2",
    "market": "0x56a208b2cc5fd14a3b1b8881e3663b9f49d79f21c0e265b00fe2791f4c1f5db4",
    "asset_id": "75464127756679880077495712957271018976146577332221108274163666841784329798122",
    "side": "BUY",
    "size": "54",
    "fee_rate_bps": "0",
    "price": "0.64",
    "status": "MATCHED",
    "match_time": "1777477273",
    "last_update": "1777477273",
    "outcome": "Down",
    "owner": "141bdcb5-a21d-811c-3e2d-8d487f6e7bfd",
    "trade_owner": "141bdcb5-a21d-811c-3e2d-8d487f6e7bfd",
    "maker_address": "0x927f7694dE44d19A72Bce76254E628d1C141D215",
    "transaction_hash": "0xadbf353b5155cccfb0228c4661d90490a6a55133ff3b3384be112b1e65091f2f",
    "bucket_index": 0,
    "maker_orders": [
        {
            "order_id": "0x3e527db52efe2a0192a0237a16e210bd1670787a0a44aa03e28782b15af9287f",
            "owner": "bd13b142-eaf0-578a-3b4b-6860cc1d3f8b",
            "maker_address": "0xB343c03eE55a50C1beC6A37A81Cfb4349656097D",
            "matched_amount": "20",
            "price": "0.37",
            "fee_rate_bps": "",
            "asset_id": "10765894100428851987409758832039533005047925107391989144133315229152059466569",
            "outcome": "Up",
            "outcome_index": 0,
            "side": "BUY"
        },
        {
            "order_id": "0xe89b37fc3d1aeec42d7d864df4fa7b5acbb7e6493ab5140788c950315f1e5257",
            "owner": "a280fa42-9047-b100-00d3-669361e9966a",
            "maker_address": "0x7629BD40d8c28DA3b784b7064308ADD8A7EA45eE",
            "matched_amount": "5",
            "price": "0.37",
            "fee_rate_bps": "",
            "asset_id": "10765894100428851987409758832039533005047925107391989144133315229152059466569",
            "outcome": "Up",
            "outcome_index": 0,
            "side": "BUY"
        },
        {
            "order_id": "0x04a3e74c6f54dd0bc345b02bd3317158148dd2833479888e2bceb05751b594bd",
            "owner": "e66e077a-770d-4465-fed3-29acbe298c21",
            "maker_address": "0x9cd2f901DAD6e203179D6e4DB4fD1F691a6F1d66",
            "matched_amount": "9",
            "price": "0.36",
            "fee_rate_bps": "",
            "asset_id": "10765894100428851987409758832039533005047925107391989144133315229152059466569",
            "outcome": "Up",
            "outcome_index": 0,
            "side": "BUY"
        },
        {
            "order_id": "0x8158f74fe1072c7c23c79aa079780f00835a84a00912e0171880a6c5d56db6b9",
            "owner": "5abfc508-cc54-e8b0-7f20-1cf68f710262",
            "maker_address": "0xd31ad544838e37D46db7730adF577053DbA05f51",
            "matched_amount": "5",
            "price": "0.35",
            "fee_rate_bps": "",
            "asset_id": "10765894100428851987409758832039533005047925107391989144133315229152059466569",
            "outcome": "Up",
            "outcome_index": 0,
            "side": "BUY"
        },
        {
            "order_id": "0x5afef27edbcaf16cb92271f945d7c9819954c93f760ce358408a26e1ba955d16",
            "owner": "614db9ff-874b-581b-60b3-e264e5fa4802",
            "maker_address": "0x0f71db1628919094a46a3adAb93AD844F24534a2",
            "matched_amount": "5",
            "price": "0.35",
            "fee_rate_bps": "",
            "asset_id": "10765894100428851987409758832039533005047925107391989144133315229152059466569",
            "outcome": "Up",
            "outcome_index": 0,
            "side": "BUY"
        },
        {
            "order_id": "0x8a0f0dd777d180f4912502b129a4189f9719523fe2f3ebf16915e3e33b0836be",
            "owner": "fc1a0cfb-5a0b-941d-06bc-6ab517bf2d6d",
            "maker_address": "0x4472ABAd726D595E936C45Bd60B1b13F6Bdc4FbA",
            "matched_amount": "5",
            "price": "0.34",
            "fee_rate_bps": "",
            "asset_id": "10765894100428851987409758832039533005047925107391989144133315229152059466569",
            "outcome": "Up",
            "outcome_index": 0,
            "side": "BUY"
        },
        {
            "order_id": "0xfd28b1436eb4567116fd6dca10e3a06f23088ff7c80c476910dab93e3d5dc1ec",
            "owner": "142a6721-2a11-2137-4afe-1046c2845778",
            "maker_address": "0x52D94dbAB16167b981D5907d2D94B9D7034e9a41",
            "matched_amount": "5",
            "price": "0.34",
            "fee_rate_bps": "",
            "asset_id": "10765894100428851987409758832039533005047925107391989144133315229152059466569",
            "outcome": "Up",
            "outcome_index": 0,
            "side": "BUY"
        }
    ],
    "trader_side": "MAKER",
    "timestamp": "1777477273602",
    "event_type": "trade"
}`

func TestWSTradeUnmarshalFromBJSON(t *testing.T) {
	var trade WSTrade
	if err := json.Unmarshal([]byte(rawTradeJSON), &trade); err != nil {
		t.Fatalf("unmarshal WSTrade failed: %v", err)
	}

	if trade.EventType != "trade" {
		t.Fatalf("unexpected event_type: got=%s want=%s", trade.EventType, "trade")
	}
	if trade.Id != "55beb29d-e2d6-4e49-a6ca-9e1316596de1" {
		t.Fatalf("unexpected id: got=%s", trade.Id)
	}
	if trade.TakerOrderId != "0x0b416f20bfbd51176520950c16a6188f306c1b12abe47b531c6ba7468e2dfef2" {
		t.Fatalf("unexpected taker_order_id: got=%s", trade.TakerOrderId)
	}
	if trade.Matchtime != 1777477273 {
		t.Fatalf("unexpected match_time: got=%d want=%d", trade.Matchtime, int64(1777477273))
	}
	if trade.Timestamp != 1777477273602 {
		t.Fatalf("unexpected timestamp: got=%d want=%d", trade.Timestamp, int64(1777477273602))
	}
	if len(trade.MakerOrders) != 7 {
		t.Fatalf("unexpected maker_orders length: got=%d want=%d", len(trade.MakerOrders), 7)
	}
	if float64(trade.FeeRateBps) != 0 {
		t.Fatalf("unexpected fee_rate_bps: got=%v want=0", trade.FeeRateBps)
	}
	if float64(trade.MakerOrders[0].FeeRateBps) != 0 {
		t.Fatalf("unexpected maker_orders[0].fee_rate_bps: got=%v want=0", trade.MakerOrders[0].FeeRateBps)
	}
}

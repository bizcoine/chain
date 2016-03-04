package fedtest

import (
	"testing"

	"chain/fedchain/bc"
	"chain/fedchain/hdkey"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
	"chain/testutil"
)

type TestDest struct {
	PrivKey                *hdkey.XKey
	PKScript, RedeemScript []byte
}

func Dest(t testing.TB) *TestDest {
	var priv *hdkey.XKey
	_, priv, err := hdkey.New()
	if err != nil {
		testutil.FatalErr(t, err)
	}

	pk, redeem, err := hdkey.Scripts([]*hdkey.XKey{priv}, nil, 1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return &TestDest{
		PrivKey:      priv,
		PKScript:     pk,
		RedeemScript: redeem,
	}
}

func (d *TestDest) Sign(t testing.TB, tx *bc.TxData, index int, assetAmount bc.AssetAmount) {
	hash := tx.HashForSig(index, assetAmount, bc.SigHashAll)

	ecPriv, err := d.PrivKey.ECPrivKey()
	if err != nil {
		testutil.FatalErr(t, err)
	}

	sig, err := ecPriv.Sign(hash[:])
	if err != nil {
		testutil.FatalErr(t, err)
	}
	der := append(sig.Serialize(), byte(bc.SigHashAll))

	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_FALSE)
	builder.AddData(der)
	builder.AddData(d.RedeemScript)
	tx.Inputs[index].SignatureScript, err = builder.Script()
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

type TestAsset struct {
	bc.AssetID
	TestDest
}

func Asset(t testing.TB) *TestAsset {
	dest := Dest(t)
	assetID := bc.ComputeAssetID(dest.PKScript, bc.Hash{})

	return &TestAsset{
		AssetID:  assetID,
		TestDest: *dest,
	}
}

func Issue(t testing.TB, asset *TestAsset, dest *TestDest, amount uint64) (*bc.Tx, *TestAsset, *TestDest) {
	if asset == nil {
		asset = Asset(t)
	}
	if dest == nil {
		dest = Dest(t)
	}
	tx := &bc.TxData{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{{
			Previous: bc.Outpoint{Index: bc.InvalidOutputIndex},
		}},
		Outputs: []*bc.TxOutput{{
			Script: dest.PKScript,
			AssetAmount: bc.AssetAmount{
				AssetID: asset.AssetID,
				Amount:  amount,
			},
		}},
	}
	asset.Sign(t, tx, 0, bc.AssetAmount{})

	return bc.NewTx(*tx), asset, dest
}

func Transfer(t testing.TB, out *state.Output, from, to *TestDest) *bc.Tx {
	tx := &bc.TxData{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{{
			Previous: out.Outpoint,
		}},
		Outputs: []*bc.TxOutput{{
			Script:      to.PKScript,
			AssetAmount: out.AssetAmount,
		}},
	}
	from.Sign(t, tx, 0, out.AssetAmount)

	return bc.NewTx(*tx)
}
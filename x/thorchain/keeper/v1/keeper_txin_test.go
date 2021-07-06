package keeperv1

import (
	. "gopkg.in/check.v1"
)

type KeeperTxInSuite struct{}

var _ = Suite(&KeeperTxInSuite{})

func (s *KeeperTxInSuite) TestTxInVoter(c *C) {
	ctx, k := setupKeeperForTest(c)

	tx := GetRandomTx()
	voter := NewObservedTxVoter(
		tx.ID,
		ObservedTxs{NewObservedTx(tx, 12, GetRandomPubKey())},
	)

	k.SetObservedTxInVoter(ctx, voter)
	voter, err := k.GetObservedTxInVoter(ctx, voter.TxID)
	c.Assert(err, IsNil)
	c.Check(voter.TxID.Equals(tx.ID), Equals, true)

	voterOut, err := k.GetObservedTxOutVoter(ctx, voter.TxID)
	c.Assert(err, IsNil)
	c.Assert(voterOut.TxID.Equals(tx.ID), Equals, true)
	c.Assert(voterOut.Tx.IsEmpty(), Equals, true)

	voter1 := NewObservedTxVoter(
		tx.ID,
		ObservedTxs{
			NewObservedTx(tx, 12, GetRandomPubKey()),
		},
	)
	k.SetObservedTxOutVoter(ctx, voter1)

	voterOut1, err := k.GetObservedTxOutVoter(ctx, voter1.TxID)
	c.Assert(err, IsNil)
	c.Assert(voterOut1.TxID.Equals(tx.ID), Equals, true)
	c.Check(voterOut1.Txs, HasLen, 1)

	// ensure that if the voter doesn't exist, we DON'T error
	tx = GetRandomTx()
	voter, err = k.GetObservedTxInVoter(ctx, tx.ID)
	c.Assert(err, IsNil)
	c.Check(voter.TxID.Equals(tx.ID), Equals, true)

	iter := k.GetObservedTxInVoterIterator(ctx)
	c.Check(iter, NotNil)
	iter.Close()

	iter1 := k.GetObservedTxOutVoterIterator(ctx)
	c.Check(iter1, NotNil)
	iter1.Close()
}

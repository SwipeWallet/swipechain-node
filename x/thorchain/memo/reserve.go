package thorchain

type ReserveMemo struct {
	MemoBase
}

func NewReserveMemo() ReserveMemo {
	return ReserveMemo{
		MemoBase: MemoBase{TxType: TxReserve},
	}
}

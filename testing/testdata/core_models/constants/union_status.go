package constants

type UnionStatus int

const (
	UNION_STATUS_UNKNOWN    UnionStatus = 0
	UNION_STATUS_ACTIVE     UnionStatus = 1
	UNION_STATUS_INACTIVE   UnionStatus = 2
	UNION_STATUS_RETIRED    UnionStatus = 3
	UNION_STATUS_RESIGNED   UnionStatus = 4
	// Smaller custom ones
	UNION_STATUS_ASSOCIATE UnionStatus = 5
	UNION_STATUS_FOP       UnionStatus = 6
	UNION_STATUS_TERM      UnionStatus = 7
)

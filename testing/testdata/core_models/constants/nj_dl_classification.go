package constants

type NJDLClassification int

const (
	NJ_DL_LAW_ENFORCEMENT_OFFICER       NJDLClassification = iota + 1 // Law Enforcement Officer
	NJ_DL_CHILD_PROTECTIVE_INVESTIGATOR                               // Child Protective Investigator
	NJ_DL_JUDICIAL_OFFICER                                            // Judicial Officer
	NJ_DL_PROSECUTOR                                                  // Prosecutor
	NJ_DL_FAMILY_MEMBER                                               // Family Member
)

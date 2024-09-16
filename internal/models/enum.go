package models

type TenderStatus string
type BidStatus string
type ServiceType string
type AuthorType string
type DecisionType string

const (
	TenderCreated   TenderStatus = "Created"
	TenderPublished TenderStatus = "Published"
	TenderClosed    TenderStatus = "Closed"
)

const (
	BidCreated   BidStatus = "Created"
	BidPublished BidStatus = "Published"
	BidCanceled  BidStatus = "Canceled"
)

const (
	Construction ServiceType = "Construction"
	Delivery     ServiceType = "Delivery"
	Manufacture  ServiceType = "Manufacture"
)

const (
	User         AuthorType = "User"
	Organization AuthorType = "Organization"
)

const (
	Approved DecisionType = "Approved"
	Rejected DecisionType = "Rejected"
)

func StrToTenderStatus(s string) (TenderStatus, error) {
	st := TenderStatus(s)
	switch st {
	case TenderCreated, TenderPublished, TenderClosed:
		return st, nil
	default:
		return st, NewParseError("unknown tender status")
	}
}

func (s *TenderStatus) UnmarshalJSON(data []byte) error {
	n := len(data)
	if n == 0 {
		return NewParseError("unknown service type")
	}

	tmp, err := StrToTenderStatus(string(data[1 : n-1]))
	if err != nil {
		return err
	}

	*s = tmp
	return nil
}

func StrToBidStatus(s string) (BidStatus, error) {
	st := BidStatus(s)
	switch st {
	case BidCreated, BidPublished, BidCanceled:
		return st, nil
	default:
		return st, NewParseError("unknown bid status")
	}
}

func (s *BidStatus) UnmarshalJSON(data []byte) error {
	n := len(data)
	if n == 0 {
		return NewParseError("unknown service type")
	}

	tmp, err := StrToBidStatus(string(data[1 : n-1]))
	if err != nil {
		return err
	}

	*s = tmp
	return nil
}

func StrToServiceType(s string) (ServiceType, error) {
	t := ServiceType(s)
	switch t {
	case Construction, Delivery, Manufacture:
		return t, nil
	default:
		return t, NewParseError("unknown service type")
	}
}

func (t *ServiceType) UnmarshalJSON(data []byte) error {
	n := len(data)
	if n == 0 {
		return NewParseError("unknown service type")
	}

	tmp, err := StrToServiceType(string(data[1 : n-1]))
	if err != nil {
		return err
	}

	*t = tmp
	return nil
}

func StrToAuthorType(s string) (AuthorType, error) {
	a := AuthorType(s)
	switch a {
	case User, Organization:
		return a, nil
	default:
		return a, NewParseError("unknown author type")
	}
}

func (a *AuthorType) UnmarshalJSON(data []byte) error {
	n := len(data)
	if n == 0 {
		return NewParseError("unknown service type")
	}

	tmp, err := StrToAuthorType(string(data[1 : n-1]))
	if err != nil {
		return err
	}

	*a = tmp
	return nil
}

func StrToDecision(s string) (DecisionType, error) {
	d := DecisionType(s)
	switch d {
	case Approved, Rejected:
		return d, nil
	default:
		return d, NewParseError("unknown author type")
	}
}

package domain

import "time"

type CommerceProduct struct {
	ID              ID        `json:"id"`
	OrganizationID  ID        `json:"-"`
	UnitID          ID        `json:"unit_id"`
	Name            string    `json:"name"`
	SKU             string    `json:"sku,omitempty"`
	Status          string    `json:"status"`
	PriceCents      int       `json:"price_cents"`
	CreatedByUserID ID        `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CommerceSaleItem struct {
	ID             ID     `json:"id"`
	ProductID      ID     `json:"product_id"`
	ProductName    string `json:"product_name"`
	SKU            string `json:"sku,omitempty"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
	TotalCents     int    `json:"total_cents"`
}

type CommerceSalePayment struct {
	ID          ID        `json:"id"`
	Method      string    `json:"method"`
	AmountCents int       `json:"amount_cents"`
	OccurredAt  time.Time `json:"occurred_at"`
	ActorUserID ID        `json:"-"`
	Reference   string    `json:"reference"`
	CreatedAt   time.Time `json:"created_at"`
}

type CommerceSale struct {
	ID              ID                  `json:"id"`
	OrganizationID  ID                  `json:"-"`
	UnitID          ID                  `json:"unit_id"`
	PersonID        *ID                 `json:"person_id,omitempty"`
	PersonName      string              `json:"person_name,omitempty"`
	Status          string              `json:"status"`
	TotalCents      int                 `json:"total_cents"`
	Currency        string              `json:"currency"`
	Note            string              `json:"note,omitempty"`
	CreatedByUserID ID                  `json:"-"`
	SoldAt          time.Time           `json:"sold_at"`
	CreatedAt       time.Time           `json:"created_at"`
	Items           []CommerceSaleItem  `json:"items"`
	Payment         CommerceSalePayment `json:"payment"`
}

type CommerceCommand struct {
	OrganizationID, UnitID, ResultID     ID
	Command, IdempotencyKey, Fingerprint string
	CreatedAt                            time.Time
}

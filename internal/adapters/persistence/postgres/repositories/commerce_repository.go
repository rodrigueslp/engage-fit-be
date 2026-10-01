package repositories

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
	portrepo "boxengage/backend/internal/ports/repositories"
)

type commerceProductRow struct {
	ID, OrganizationID, UnitID, Name, SKU, Status, CreatedByUserID string
	PriceCents                                                     int
	CreatedAt, UpdatedAt                                           time.Time
}

func (r CommerceGormRepository) UnitOrganization(ctx context.Context, unitID domain.ID) (domain.ID, error) {
	var organizationID string
	result := databaseForContext(r.db, ctx).Raw(`SELECT organization_id FROM boxes WHERE id = ? AND status = 'active'`, stringID(unitID)).Scan(&organizationID)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 || organizationID == "" {
		return "", portrepo.ErrCommerceNotFound
	}
	return domain.ID(organizationID), nil
}

func (r CommerceGormRepository) ListProducts(ctx context.Context, organizationID, unitID domain.ID, includeInactive bool) ([]domain.CommerceProduct, error) {
	query := `SELECT id, organization_id, unit_id, name, COALESCE(sku, '') AS sku, price_cents, status, created_by_user_id, created_at, updated_at FROM commerce_products WHERE organization_id = ? AND unit_id = ?`
	args := []any{stringID(organizationID), stringID(unitID)}
	if !includeInactive {
		query += ` AND status = 'active'`
	}
	query += ` ORDER BY CASE status WHEN 'active' THEN 0 ELSE 1 END, name`
	var rows []commerceProductRow
	if err := databaseForContext(r.db, ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	values := make([]domain.CommerceProduct, 0, len(rows))
	for _, row := range rows {
		values = append(values, mapCommerceProduct(row))
	}
	return values, nil
}

func (r CommerceGormRepository) GetProduct(ctx context.Context, organizationID, unitID, productID domain.ID) (*domain.CommerceProduct, error) {
	var row commerceProductRow
	result := databaseForContext(r.db, ctx).Raw(`SELECT id, organization_id, unit_id, name, COALESCE(sku, '') AS sku, price_cents, status, created_by_user_id, created_at, updated_at FROM commerce_products WHERE organization_id = ? AND unit_id = ? AND id = ?`, stringID(organizationID), stringID(unitID), stringID(productID)).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 || row.ID == "" {
		return nil, portrepo.ErrCommerceNotFound
	}
	value := mapCommerceProduct(row)
	return &value, nil
}

func (r CommerceGormRepository) GetProductsForSale(ctx context.Context, organizationID, unitID domain.ID, ids []domain.ID) ([]domain.CommerceProduct, error) {
	stringsIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		stringsIDs = append(stringsIDs, stringID(id))
	}
	var rows []commerceProductRow
	err := databaseForContext(r.db, ctx).Raw(`SELECT id, organization_id, unit_id, name, COALESCE(sku, '') AS sku, price_cents, status, created_by_user_id, created_at, updated_at FROM commerce_products WHERE organization_id = ? AND unit_id = ? AND status = 'active' AND id IN ? ORDER BY id`, stringID(organizationID), stringID(unitID), stringsIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	values := make([]domain.CommerceProduct, 0, len(rows))
	for _, row := range rows {
		values = append(values, mapCommerceProduct(row))
	}
	return values, nil
}

func (r CommerceGormRepository) CreateProduct(ctx context.Context, value *domain.CommerceProduct) error {
	err := databaseForContext(r.db, ctx).Exec(`INSERT INTO commerce_products (id, organization_id, unit_id, name, sku, price_cents, status, created_by_user_id, created_at, updated_at) VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?)`, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID), value.Name, value.SKU, value.PriceCents, value.Status, stringID(value.CreatedByUserID), value.CreatedAt, value.UpdatedAt).Error
	if isUniqueViolation(err) {
		return portrepo.ErrCommerceConflict
	}
	return err
}

func (r CommerceGormRepository) UpdateProduct(ctx context.Context, value *domain.CommerceProduct) error {
	result := databaseForContext(r.db, ctx).Exec(`UPDATE commerce_products SET name = ?, sku = NULLIF(?, ''), price_cents = ?, status = ?, updated_at = ? WHERE id = ? AND organization_id = ? AND unit_id = ?`, value.Name, value.SKU, value.PriceCents, value.Status, value.UpdatedAt, stringID(value.ID), stringID(value.OrganizationID), stringID(value.UnitID))
	if isUniqueViolation(result.Error) {
		return portrepo.ErrCommerceConflict
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return portrepo.ErrCommerceNotFound
	}
	return nil
}

type commerceSaleRow struct {
	ID, OrganizationID, UnitID, PersonID, PersonName, Status, Currency, Note, CreatedByUserID string
	TotalCents                                                                                int
	SoldAt, CreatedAt                                                                         time.Time
}

func (r CommerceGormRepository) ListSales(ctx context.Context, organizationID, unitID domain.ID, limit int) ([]domain.CommerceSale, error) {
	var rows []commerceSaleRow
	err := databaseForContext(r.db, ctx).Raw(commerceSaleSelect+` WHERE s.organization_id = ? AND s.unit_id = ? ORDER BY s.sold_at DESC LIMIT ?`, stringID(organizationID), stringID(unitID), limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	values := make([]domain.CommerceSale, 0, len(rows))
	for _, row := range rows {
		value, err := r.loadSale(ctx, row)
		if err != nil {
			return nil, err
		}
		values = append(values, *value)
	}
	return values, nil
}

func (r CommerceGormRepository) ListPrivacy(ctx context.Context, unitID, personID domain.ID) ([]domain.CommerceSale, error) {
	var rows []commerceSaleRow
	err := databaseForContext(r.db, ctx).Raw(commerceSaleSelect+` WHERE s.unit_id = ? AND s.person_id = ? ORDER BY s.sold_at DESC`, stringID(unitID), stringID(personID)).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	values := make([]domain.CommerceSale, 0, len(rows))
	for _, row := range rows {
		value, err := r.loadSale(ctx, row)
		if err != nil {
			return nil, err
		}
		values = append(values, *value)
	}
	return values, nil
}

func (r CommerceGormRepository) GetSale(ctx context.Context, organizationID, unitID, saleID domain.ID) (*domain.CommerceSale, error) {
	var row commerceSaleRow
	result := databaseForContext(r.db, ctx).Raw(commerceSaleSelect+` WHERE s.organization_id = ? AND s.unit_id = ? AND s.id = ?`, stringID(organizationID), stringID(unitID), stringID(saleID)).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 || row.ID == "" {
		return nil, portrepo.ErrCommerceNotFound
	}
	return r.loadSale(ctx, row)
}

const commerceSaleSelect = `SELECT s.id, s.organization_id, s.unit_id, COALESCE(s.person_id::text, '') AS person_id, COALESCE(p.preferred_name, p.name, '') AS person_name, s.status, s.total_cents, s.currency, COALESCE(s.note, '') AS note, s.created_by_user_id, s.sold_at, s.created_at FROM commerce_sales s LEFT JOIN persons p ON p.id = s.person_id AND p.organization_id = s.organization_id`

func (r CommerceGormRepository) loadSale(ctx context.Context, row commerceSaleRow) (*domain.CommerceSale, error) {
	value := &domain.CommerceSale{ID: domain.ID(row.ID), OrganizationID: domain.ID(row.OrganizationID), UnitID: domain.ID(row.UnitID), PersonName: row.PersonName, Status: row.Status, TotalCents: row.TotalCents, Currency: row.Currency, Note: row.Note, CreatedByUserID: domain.ID(row.CreatedByUserID), SoldAt: row.SoldAt, CreatedAt: row.CreatedAt, Items: []domain.CommerceSaleItem{}}
	if row.PersonID != "" {
		id := domain.ID(row.PersonID)
		value.PersonID = &id
	}
	type itemRow struct {
		ID, ProductID, ProductName, SKU      string
		Quantity, UnitPriceCents, TotalCents int
	}
	var items []itemRow
	db := databaseForContext(r.db, ctx)
	if err := db.Raw(`SELECT id, product_id, product_name, COALESCE(sku, '') AS sku, quantity, unit_price_cents, total_cents FROM commerce_sale_items WHERE sale_id = ? ORDER BY product_name`, row.ID).Scan(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		value.Items = append(value.Items, domain.CommerceSaleItem{ID: domain.ID(item.ID), ProductID: domain.ID(item.ProductID), ProductName: item.ProductName, SKU: item.SKU, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents, TotalCents: item.TotalCents})
	}
	type paymentRow struct {
		ID, Method, ActorUserID, Reference string
		AmountCents                        int
		OccurredAt, CreatedAt              time.Time
	}
	var payment paymentRow
	result := db.Raw(`SELECT id, method, amount_cents, occurred_at, actor_user_id, reference, created_at FROM commerce_sale_payments WHERE sale_id = ?`, row.ID).Scan(&payment)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, portrepo.ErrCommerceNotFound
	}
	value.Payment = domain.CommerceSalePayment{ID: domain.ID(payment.ID), Method: payment.Method, AmountCents: payment.AmountCents, OccurredAt: payment.OccurredAt, ActorUserID: domain.ID(payment.ActorUserID), Reference: payment.Reference, CreatedAt: payment.CreatedAt}
	return value, nil
}

func (r CommerceGormRepository) CreateSale(ctx context.Context, sale *domain.CommerceSale) error {
	db := databaseForContext(r.db, ctx)
	var person any
	if sale.PersonID != nil {
		person = stringID(*sale.PersonID)
	}
	if err := db.Exec(`INSERT INTO commerce_sales (id, organization_id, unit_id, person_id, status, total_cents, currency, note, created_by_user_id, sold_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?)`, stringID(sale.ID), stringID(sale.OrganizationID), stringID(sale.UnitID), person, sale.Status, sale.TotalCents, sale.Currency, sale.Note, stringID(sale.CreatedByUserID), sale.SoldAt, sale.CreatedAt).Error; err != nil {
		return err
	}
	for _, item := range sale.Items {
		if err := db.Exec(`INSERT INTO commerce_sale_items (id, sale_id, organization_id, unit_id, product_id, product_name, sku, quantity, unit_price_cents, total_cents, created_at) VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`, stringID(item.ID), stringID(sale.ID), stringID(sale.OrganizationID), stringID(sale.UnitID), stringID(item.ProductID), item.ProductName, item.SKU, item.Quantity, item.UnitPriceCents, item.TotalCents, sale.CreatedAt).Error; err != nil {
			return err
		}
	}
	payment := sale.Payment
	return db.Exec(`INSERT INTO commerce_sale_payments (id, sale_id, organization_id, unit_id, method, amount_cents, occurred_at, actor_user_id, reference, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, stringID(payment.ID), stringID(sale.ID), stringID(sale.OrganizationID), stringID(sale.UnitID), payment.Method, payment.AmountCents, payment.OccurredAt, stringID(payment.ActorUserID), payment.Reference, payment.CreatedAt).Error
}

func (r CommerceGormRepository) LockCommand(ctx context.Context, organizationID, unitID domain.ID, command, key string) error {
	return databaseForContext(r.db, ctx).Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?, 0))`, stringID(organizationID)+":"+stringID(unitID)+":"+command+":"+key).Error
}

func (r CommerceGormRepository) FindCommand(ctx context.Context, organizationID, unitID domain.ID, command, key string) (*domain.CommerceCommand, error) {
	type row struct {
		OrganizationID, UnitID, Command, IdempotencyKey, Fingerprint, ResultID string
		CreatedAt                                                              time.Time
	}
	var value row
	result := databaseForContext(r.db, ctx).Raw(`SELECT organization_id, unit_id, command, idempotency_key, fingerprint, result_id, created_at FROM commerce_command_requests WHERE organization_id = ? AND unit_id = ? AND command = ? AND idempotency_key = ?`, stringID(organizationID), stringID(unitID), command, key).Scan(&value)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &domain.CommerceCommand{OrganizationID: domain.ID(value.OrganizationID), UnitID: domain.ID(value.UnitID), Command: value.Command, IdempotencyKey: value.IdempotencyKey, Fingerprint: value.Fingerprint, ResultID: domain.ID(value.ResultID), CreatedAt: value.CreatedAt}, nil
}

func (r CommerceGormRepository) CreateCommand(ctx context.Context, value *domain.CommerceCommand) error {
	err := databaseForContext(r.db, ctx).Exec(`INSERT INTO commerce_command_requests (organization_id, unit_id, command, idempotency_key, fingerprint, result_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, stringID(value.OrganizationID), stringID(value.UnitID), value.Command, value.IdempotencyKey, value.Fingerprint, stringID(value.ResultID), value.CreatedAt).Error
	if isUniqueViolation(err) {
		return portrepo.ErrCommerceConflict
	}
	return err
}

func mapCommerceProduct(row commerceProductRow) domain.CommerceProduct {
	return domain.CommerceProduct{ID: domain.ID(row.ID), OrganizationID: domain.ID(row.OrganizationID), UnitID: domain.ID(row.UnitID), Name: row.Name, SKU: row.SKU, Status: row.Status, PriceCents: row.PriceCents, CreatedByUserID: domain.ID(row.CreatedByUserID), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

var _ portrepo.CommerceRepository = CommerceGormRepository{}

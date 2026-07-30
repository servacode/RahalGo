package orders

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

const orderSelect = `
	SELECT o.id, o.number, o.customer_id, cu.phone, cu.full_name,
	       o.merchant_id, mr.name, o.driver_id, dr.phone,
	       o.status, o.address_text,
	       ST_Y(o.dropoff::geometry), ST_X(o.dropoff::geometry),
	       o.zone_id, z.name,
	       o.payment_method, o.subtotal, o.delivery_fee, o.discount, o.total,
	       o.wallet_paid, o.cash_due, o.promo_code, o.notes, o.cancel_reason, o.created_at
	FROM orders o
	JOIN users cu ON cu.id = o.customer_id
	JOIN merchants mr ON mr.id = o.merchant_id
	LEFT JOIN users dr ON dr.id = o.driver_id
	LEFT JOIN delivery_zones z ON z.id = o.zone_id`

func scanOrder(row pgx.Row) (*Order, error) {
	var o Order
	err := row.Scan(&o.ID, &o.Number, &o.CustomerID, &o.CustomerPhone, &o.CustomerName,
		&o.MerchantID, &o.MerchantName, &o.DriverID, &o.DriverPhone,
		&o.Status, &o.AddressText, &o.Lat, &o.Lng, &o.ZoneID, &o.ZoneName,
		&o.PaymentMethod, &o.Subtotal, &o.DeliveryFee, &o.Discount, &o.Total,
		&o.WalletPaid, &o.CashDue, &o.PromoCode, &o.Notes, &o.CancelReason, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Order, error) {
	o, err := scanOrder(s.db.QueryRow(ctx, orderSelect+` WHERE o.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, menu_item_id, name, unit_price, qty, note, options
		FROM order_items WHERE order_id = $1`, id)
	if err != nil {
		return nil, err
	}
	o.Items = []OrderItem{}
	for rows.Next() {
		var it OrderItem
		var opts []byte
		if err := rows.Scan(&it.ID, &it.MenuItemID, &it.Name, &it.UnitPrice, &it.Qty, &it.Note, &opts); err != nil {
			rows.Close()
			return nil, err
		}
		it.Options = []OptionSnapshot{}
		_ = json.Unmarshal(opts, &it.Options)
		o.Items = append(o.Items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	eRows, err := s.db.Query(ctx, `
		SELECT from_status, to_status, actor_id, note, created_at
		FROM order_events WHERE order_id = $1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer eRows.Close()
	o.Events = []Event{}
	for eRows.Next() {
		var e Event
		if err := eRows.Scan(&e.FromStatus, &e.ToStatus, &e.ActorID, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		o.Events = append(o.Events, e)
	}
	if err := eRows.Err(); err != nil {
		return nil, err
	}
	o.Rating = s.ratingFor(ctx, id)
	return o, nil
}

type ListFilter struct {
	Status     string
	MerchantID string
	CustomerID string
	DriverID   string
	Query      string // رقم طلب أو هاتف زبون
	OpenOnly   bool   // الطلبات الجارية فقط
	Page       int
	PerPage    int
}

func (s *Service) List(ctx context.Context, f ListFilter) (*OrderPage, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 || f.PerPage > 100 {
		f.PerPage = 20
	}
	where := ` WHERE ($1 = '' OR o.status = $1)
		AND ($2 = '' OR o.merchant_id::text = $2)
		AND ($3 = '' OR o.customer_id::text = $3)
		AND ($4 = '' OR o.driver_id::text = $4)
		AND ($5 = '' OR o.number::text = $5 OR cu.phone ILIKE '%'||$5||'%')
		AND (NOT $6 OR o.closed_at IS NULL)`

	var total int
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM orders o JOIN users cu ON cu.id = o.customer_id`+where,
		f.Status, f.MerchantID, f.CustomerID, f.DriverID, f.Query, f.OpenOnly).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, orderSelect+where+`
		ORDER BY o.created_at DESC LIMIT $7 OFFSET $8`,
		f.Status, f.MerchantID, f.CustomerID, f.DriverID, f.Query, f.OpenOnly,
		f.PerPage, (f.Page-1)*f.PerPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &OrderPage{Orders: orders, Total: total, Page: f.Page, PerPage: f.PerPage}, nil
}

package orders_test

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestDriverView_FreeDeliveryShowsFullFee — the driver sees his full fee on a free-delivery
// order, not the customer's zero, and free_waived tells his app it is free for the customer.
func TestDriverView_FreeDeliveryShowsFullFee(t *testing.T) {
	o := &orders.Order{DeliveryFee: 0, FreeWaived: 5000}
	v := orders.ViewFor(orders.AudienceDriver, o)
	if got, _ := v["delivery_fee"].(int64); got != 5000 {
		t.Fatalf("driver delivery_fee = %v, want 5000", v["delivery_fee"])
	}
	if got, _ := v["free_waived"].(float64); got != 5000 {
		t.Fatalf("driver free_waived = %v, want 5000", v["free_waived"])
	}
	c := orders.ViewFor(orders.AudienceCustomer, o)
	if got, _ := c["delivery_fee"].(float64); got != 0 {
		t.Fatalf("customer delivery_fee = %v, want 0", c["delivery_fee"])
	}
}

package orders

import "context"

// OpenCustomChatForTest وAutoAgreeFixedFeeForTest **ما يقع لحظةَ الإسناد** — بلا وردية سائقٍ ولا موزّع.
func (s *Service) OpenCustomChatForTest(ctx context.Context, orderID, customerID, driverID string) {
	s.openCustomChat(ctx, orderID, customerID, driverID)
}

func (s *Service) AutoAgreeFixedFeeForTest(ctx context.Context, orderID, customerID, driverID string) {
	s.autoAgreeFixedFee(ctx, orderID, customerID, driverID)
}

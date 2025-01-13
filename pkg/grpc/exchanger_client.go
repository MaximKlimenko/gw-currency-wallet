package grpc

import (
	"context"

	pb "github.com/MaximKlimenko/proto-exchange/exchange"

	"google.golang.org/grpc"
)

type ExchangerClient struct {
	client pb.ExchangeServiceClient
}

func NewExchangerClient(conn *grpc.ClientConn) *ExchangerClient {
	return &ExchangerClient{
		client: pb.NewExchangeServiceClient(conn),
	}
}

func (e *ExchangerClient) GetExchangeRate(from, to string) (float64, error) {
	resp, err := e.client.GetExchangeRateForCurrency(context.Background(), &pb.CurrencyRequest{
		FromCurrency: from,
		ToCurrency:   to,
	})
	if err != nil {
		return 0, err
	}
	return float64(resp.Rate), nil
}

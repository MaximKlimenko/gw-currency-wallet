package exchanger

import (
	"context"
	"fmt"

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

func (e *ExchangerClient) GetExchangeRates() (map[string]float32, error) {
	// Выполняем gRPC-запрос с указанной валютой
	resp, err := e.client.GetExchangeRates(context.Background(), &pb.Empty{})
	if err != nil {
		fmt.Println("Ошибка при запросе курсов валют:", err)
		return nil, err
	}

	// Возвращаем курсы валют
	return resp.Rates, nil
}

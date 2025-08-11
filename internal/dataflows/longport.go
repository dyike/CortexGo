package dataflows

import (
	"context"
	"errors"
	"log"

	"github.com/longportapp/openapi-go/config"
	"github.com/longportapp/openapi-go/quote"
	"github.com/longportapp/openapi-go/trade"
)

type LongportClient struct {
	tradeCtx *trade.TradeContext
	quoteCtx *quote.QuoteContext
}

func NewLongportClient() *LongportClient {
	conf, err := config.New(config.WithConfigKey("YOUR_APP_KEY", "YOUR_APP_SECRET", "YOUR_ACCESS_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	tradeContext, err := trade.NewFromCfg(conf)
	if err != nil {
		log.Fatal(err)
	}

	quoteContext, err := quote.NewFromCfg(conf)
	if err != nil {
		log.Fatal(err)
	}
	return &LongportClient{
		tradeCtx: tradeContext,
		quoteCtx: quoteContext,
	}
}

func (lpc *LongportClient) GetStaticInfo(ctx context.Context, symbols []string) (staticInfos []*quote.StaticInfo, err error) {
	if lpc.quoteCtx != nil {
		return lpc.quoteCtx.StaticInfo(ctx, symbols)
	}
	return nil, errors.New("quote context is nil")
}

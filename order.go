package shopify

import (
	"context"
	"fmt"
	"strings"

	"github.com/r0busta/graphql"
	"github.com/spl0i7/go-shopify-graphql-model/v4/graph/model"
)

//go:generate mockgen -destination=./mock/order_service.go -package=mock . OrderService
type OrderService interface {
	Get(ctx context.Context, id graphql.ID) (*model.Order, error)

	List(ctx context.Context, opts ListOptions) ([]model.Order, error)
	ListAll(ctx context.Context) ([]model.Order, error)

	ListAfterCursor(ctx context.Context, opts ListOptions) ([]model.Order, *string, *string, error)

	Update(ctx context.Context, input model.OrderInput) error
}

type OrderServiceOp struct {
	client *Client
}

var _ OrderService = &OrderServiceOp{}

type mutationOrderUpdate struct {
	OrderUpdateResult struct {
		UserErrors []model.UserError `json:"userErrors,omitempty"`
	} `graphql:"orderUpdate(input: $input)" json:"orderUpdate"`
}

const orderBaseQuery = `
	id
	legacyResourceId
	name
	createdAt
	customer{
		id
		legacyResourceId
		firstName
		displayName
		email
	}
	clientIp
	shippingAddress{
		address1
		address2
		city
		province
		country
		zip
	}
	shippingLine{
		originalPriceSet{
			presentmentMoney{
				amount
				currencyCode
			}
			shopMoney{
				amount
				currencyCode
			}
		}
		title
	}
	taxLines{
		priceSet{
			presentmentMoney{
				amount
				currencyCode
			}
			shopMoney{
				amount
				currencyCode
			}
		}
		rate
		ratePercentage
		title
	}
	totalReceivedSet{
		presentmentMoney{
			amount
			currencyCode
		}
		shopMoney{
			amount
			currencyCode
		}
	}
	note
	tags
	transactions {
		processedAt
		status
		kind
		test
		amountSet {
			shopMoney {
				amount
				currencyCode
			}
		}
	}	
`

const orderLightQuery = `
	id
	legacyResourceId
	name
	createdAt
	customer{
		id
		legacyResourceId
		firstName
		displayName
		email
	}
	shippingAddress{
		address1
		address2
		city
		province
		country
		zip
	}
	shippingLine{
		title
	}
	totalReceivedSet{
		shopMoney{
			amount
		}
	}
	note
	tags
`

const lineItemFragment = `
fragment lineItem on LineItem {
	id
	sku
	quantity
	fulfillableQuantity
	fulfillmentStatus
	product{
		id
		legacyResourceId										
	}
	vendor
	title
	variantTitle
	variant{
		id
		legacyResourceId	
		selectedOptions{
			name
			value
		}									
	}
	originalTotalSet{
		presentmentMoney{
			amount
			currencyCode
		}
		shopMoney{
			amount
			currencyCode
		}
	}
	originalUnitPriceSet{
		presentmentMoney{
			amount
			currencyCode
		}
		shopMoney{
			amount
			currencyCode
		}
	}
	discountedUnitPriceSet{
		presentmentMoney{
			amount
			currencyCode
		}
		shopMoney{
			amount
			currencyCode
		}
	}
	discountedTotalSet{
		presentmentMoney{
			amount
			currencyCode
		}
		shopMoney{
			amount
			currencyCode
		}
	}
}
`

const lineItemFragmentLight = `
fragment lineItem on LineItem {
	id
	sku
	quantity
	fulfillableQuantity
	fulfillmentStatus
	vendor
	title
	variantTitle
}
`

func (s *OrderServiceOp) Get(ctx context.Context, id graphql.ID) (*model.Order, error) {
	q := fmt.Sprintf(`
		query order($id: ID!) {
			node(id: $id){
				... on Order {
					%s
					lineItems(first:50){
						edges{
							node{
								...lineItem
							}
						}
					}
					fulfillmentOrders(first:5){
						edges {
							node {
								id
								status
								lineItems(first:50){
									edges {
										node {
											id
											remainingQuantity
											totalQuantity
											lineItem{
												sku
											}								
										}
									}
								}
							}
						}
					}					
				}
			}
		}

		%s
	`, orderBaseQuery, lineItemFragment)

	vars := map[string]interface{}{
		"id": id,
	}

	out := struct {
		Order *model.Order `json:"node"`
	}{}
	err := s.client.gql.QueryString(ctx, q, vars, &out)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	return out.Order, nil
}

func (s *OrderServiceOp) List(ctx context.Context, opts ListOptions) ([]model.Order, error) {
	q := fmt.Sprintf(`
		{
			orders(query: "$query"){
				edges{
					node{
						%s
						lineItems{
							edges{
								node{
									...lineItem
								}
							}
						}
					}
				}
			}
		}

		%s
	`, orderBaseQuery, lineItemFragment)

	q = strings.ReplaceAll(q, "$query", opts.Query)

	res := []model.Order{}
	err := s.client.BulkOperation.BulkQuery(ctx, q, &res)
	if err != nil {
		return nil, fmt.Errorf("bulk query: %w", err)
	}

	return res, nil
}

func (s *OrderServiceOp) ListAll(ctx context.Context) ([]model.Order, error) {
	q := fmt.Sprintf(`
		{
			orders(query: "$query"){
				edges{
					node{
						%s
						lineItems{
							edges{
								node{
									...lineItem
								}
							}
						}
					}
				}
			}
		}

		%s
	`, orderBaseQuery, lineItemFragment)

	res := []model.Order{}
	err := s.client.BulkOperation.BulkQuery(ctx, q, &res)
	if err != nil {
		return nil, fmt.Errorf("bulk query: %w", err)
	}

	return res, nil
}

func (s *OrderServiceOp) ListAfterCursor(ctx context.Context, opts ListOptions) ([]model.Order, *string, *string, error) {
	q := fmt.Sprintf(`
		query orders($query: String, $first: Int, $last: Int, $before: String, $after: String, $reverse: Boolean) {
			orders(query: $query, first: $first, last: $last, before: $before, after: $after, reverse: $reverse){
				edges{
					node{
						%s

						lineItems(first:25){
							edges{
								node{
									...lineItem
								}
							}
						}
					}
					cursor
				}
				pageInfo{
					hasNextPage
				}				
			}
		}

		%s
	`, orderLightQuery, lineItemFragmentLight)

	vars := map[string]interface{}{
		"query":   opts.Query,
		"reverse": opts.Reverse,
	}

	if opts.After != "" {
		vars["after"] = opts.After
	} else if opts.Before != "" {
		vars["before"] = opts.Before
	}

	if opts.First > 0 {
		vars["first"] = opts.First
	} else if opts.Last > 0 {
		vars["last"] = opts.Last
	}

	out := struct {
		Orders struct {
			Edges []struct {
				OrderQueryResult *model.Order `json:"node,omitempty"`
				Cursor           string       `json:"cursor,omitempty"`
			} `json:"edges,omitempty"`
			PageInfo struct {
				HasNextPage bool `json:"hasNextPage,omitempty"`
			} `json:"pageInfo,omitempty"`
		} `json:"orders,omitempty"`
	}{}
	err := s.client.gql.QueryString(ctx, q, vars, &out)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query: %w", err)
	}

	res := []model.Order{}
	var firstCursor *string
	var lastCursor *string
	if len(out.Orders.Edges) > 0 {
		firstCursor = &out.Orders.Edges[0].Cursor
		lastCursor = &out.Orders.Edges[len(out.Orders.Edges)-1].Cursor
		for _, o := range out.Orders.Edges {
			res = append(res, *o.OrderQueryResult)
		}
	}

	return res, firstCursor, lastCursor, nil
}

func (s *OrderServiceOp) Update(ctx context.Context, input model.OrderInput) error {
	m := mutationOrderUpdate{}

	vars := map[string]interface{}{
		"input": input,
	}
	err := s.client.gql.Mutate(ctx, &m, vars)
	if err != nil {
		return fmt.Errorf("mutation: %w", err)
	}

	if len(m.OrderUpdateResult.UserErrors) > 0 {
		return fmt.Errorf("%+v", m.OrderUpdateResult.UserErrors)
	}

	return nil
}

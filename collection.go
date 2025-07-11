package shopify

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spl0i7/go-shopify-graphql-model/v4/graph/model"
)

//go:generate mockgen -destination=./mock/collection_service.go -package=mock . CollectionService
type CollectionService interface {
	ListAll(ctx context.Context) ([]model.Collection, error)

	Get(ctx context.Context, id string) (*model.Collection, error)

	Create(ctx context.Context, collection model.CollectionInput) (*string, error)
	CreateBulk(ctx context.Context, collections []model.CollectionInput) error

	Update(ctx context.Context, collection model.CollectionInput) error
}

type CollectionServiceOp struct {
	client *Client
}

var _ CollectionService = &CollectionServiceOp{}

type mutationCollectionCreate struct {
	CollectionCreateResult struct {
		Collection *struct {
			ID string `json:"id,omitempty"`
		} `json:"collection,omitempty"`

		UserErrors []model.UserError `json:"userErrors,omitempty"`
	} `graphql:"collectionCreate(input: $input)" json:"collectionCreate"`
}

type mutationCollectionUpdate struct {
	CollectionCreateResult struct {
		UserErrors []model.UserError `json:"userErrors,omitempty"`
	} `graphql:"collectionUpdate(input: $input)" json:"collectionUpdate"`
}

var collectionQuery = `
	id
	handle	
	title

	products(first:250, after: $cursor){
		edges{
			node{
				id
			}
			cursor
		}
		pageInfo{
			hasNextPage
		}		
	}	
`

var collectionBulkQuery = `
	id
	handle	
	title
`

func (s *CollectionServiceOp) ListAll(ctx context.Context) ([]model.Collection, error) {
	q := fmt.Sprintf(`
		{
			collections{
				edges{
					node{
						%s
					}
				}
			}
		}
	`, collectionBulkQuery)

	res := []model.Collection{}
	err := s.client.BulkOperation.BulkQuery(ctx, q, &res)
	if err != nil {
		return nil, fmt.Errorf("bulk query: %w", err)
	}

	return res, nil
}

func (s *CollectionServiceOp) Get(ctx context.Context, id string) (*model.Collection, error) {
	out, err := s.getPage(ctx, id, "")
	if err != nil {
		return nil, err
	}

	nextPageData := out
	hasNextPage := out.Products.PageInfo.HasNextPage
	for hasNextPage && len(nextPageData.Products.Edges) > 0 {
		cursor := nextPageData.Products.Edges[len(nextPageData.Products.Edges)-1].Cursor
		nextPageData, err := s.getPage(ctx, id, cursor)
		if err != nil {
			return nil, err
		}
		out.Products.Edges = append(out.Products.Edges, nextPageData.Products.Edges...)
		hasNextPage = nextPageData.Products.PageInfo.HasNextPage
	}

	return out, nil
}

func (s *CollectionServiceOp) getPage(ctx context.Context, id string, cursor string) (*model.Collection, error) {
	q := fmt.Sprintf(`
		query collection($id: ID!, $cursor: String) {
			collection(id: $id){
				%s
			}
		}
	`, collectionQuery)

	vars := map[string]interface{}{
		"id": id,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	out := struct {
		Collection *model.Collection `json:"collection"`
	}{}
	err := s.client.gql.QueryString(ctx, q, vars, &out)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	return out.Collection, nil
}

func (s *CollectionServiceOp) CreateBulk(ctx context.Context, collections []model.CollectionInput) error {
	for _, c := range collections {
		_, err := s.client.Collection.Create(ctx, c)
		if err != nil {
			log.Warnf("Couldn't create collection (%v): %s", c, err)
		}
	}

	return nil
}

func (s *CollectionServiceOp) Create(ctx context.Context, collection model.CollectionInput) (*string, error) {
	m := mutationCollectionCreate{}

	vars := map[string]interface{}{
		"input": collection,
	}
	err := s.client.gql.Mutate(ctx, &m, vars)
	if err != nil {
		return nil, fmt.Errorf("mutation: %w", err)
	}

	if len(m.CollectionCreateResult.UserErrors) > 0 {
		return nil, fmt.Errorf("%+v", m.CollectionCreateResult.UserErrors)
	}

	return &m.CollectionCreateResult.Collection.ID, nil
}

func (s *CollectionServiceOp) Update(ctx context.Context, collection model.CollectionInput) error {
	m := mutationCollectionUpdate{}

	vars := map[string]interface{}{
		"input": collection,
	}
	err := s.client.gql.Mutate(ctx, &m, vars)
	if err != nil {
		return fmt.Errorf("mutation: %w", err)
	}

	if len(m.CollectionCreateResult.UserErrors) > 0 {
		return fmt.Errorf("%+v", m.CollectionCreateResult.UserErrors)
	}

	return nil
}

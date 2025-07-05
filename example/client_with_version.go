package main

import (
	"os"

	shopify "github.com/spl0i7/go-shopify-graphql/v9"
	graphqlclient "github.com/spl0i7/go-shopify-graphql/v9/graphql"
)

func clientWithVersion() *shopify.Client {
	gqlClient := graphqlclient.NewClient(os.Getenv("STORE_NAME"), graphqlclient.WithToken(os.Getenv("STORE_ACCESS_TOKEN")), graphqlclient.WithVersion("2022-10"))

	return shopify.NewClient(shopify.WithGraphQLClient(gqlClient))
}

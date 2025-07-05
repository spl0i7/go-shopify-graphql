package main

import (
	"context"
	"fmt"

	"github.com/spl0i7/go-shopify-graphql/v9"
)

func collections(client *shopify.Client) {
	// Get all collections
	collections, err := client.Collection.ListAll(context.Background())
	if err != nil {
		panic(err)
	}

	// Print out the result
	for _, c := range collections {
		fmt.Println(c.Handle)
	}
}

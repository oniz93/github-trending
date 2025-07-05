package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// MilvusConnection holds the Milvus client.
type MilvusConnection struct {
	Client         client.Client
	CollectionName string
}

// NewMilvusConnection creates a new connection to the Milvus server.
func NewMilvusConnection(host, port string) (*MilvusConnection, error) {
	addr := fmt.Sprintf("%s:%s", host, port)

	milvusClient, err := client.NewClient(context.Background(), client.Config{
		Address: addr,
		APIKey:  "", // No API key for standalone
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Milvus client: %w", err)
	}

	collectionName := "repository_embeddings"

	// Check if collection exists, if not, create it
	exists, err := milvusClient.HasCollection(context.Background(), collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to check Milvus collection existence: %w", err)
	}

	if !exists {
		log.Printf("Milvus collection '%s' does not exist, creating it...", collectionName)
		// Define schema for the collection
		schema := &entity.Schema{
			CollectionName: collectionName,
			Description:    "Repository README embeddings",
			Fields: []*entity.Field{
				{
					Name:       "repo_id",
					DataType:   entity.FieldTypeInt64,
					PrimaryKey: true,
					AutoID:     false,
				},
				{
					Name:     "embedding",
					DataType: entity.FieldTypeFloatVector,
					TypeParams: map[string]string{
						"dim": "384",
					},
				},
			},
		}
		err = milvusClient.CreateCollection(context.Background(), schema, entity.DefaultShardNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to create Milvus collection: %w", err)
		}
		log.Printf("Milvus collection '%s' created successfully.", collectionName)
	}

	// Load collection into memory
	err = milvusClient.LoadCollection(context.Background(), collectionName, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load Milvus collection: %w", err)
	}

	return &MilvusConnection{Client: milvusClient, CollectionName: collectionName}, nil
}

// InsertEmbedding inserts a repository embedding into Milvus.
func (mc *MilvusConnection) InsertEmbedding(repoID int64, embedding []float32) error {
	// Prepare columns
	repoIDs := []int64{repoID}
	embeddings := [][]float32{embedding}

	// Create columns
		idCol := entity.NewColumnInt64("repo_id", repoIDs)
	embeddingCol := entity.NewColumnFloatVector("embedding", 384, embeddings)

	// Insert data
	_, err := mc.Client.Insert(context.Background(), mc.CollectionName, "", idCol, embeddingCol)
	if err != nil {
		return fmt.Errorf("failed to insert embedding into Milvus: %w", err)
	}

	return nil
}

// SearchEmbeddings searches for similar embeddings in Milvus.
func (mc *MilvusConnection) SearchEmbeddings(vector []float32, topK int) ([]client.SearchResult, error) {
	searchParam := entity.IndexFlatSearchParam{}

	res, err := mc.Client.Search(
		context.Background(),
		mc.CollectionName,
		[]string{}, // partitions
		"",         // expr
		[]string{"repo_id"}, // outputFields
		[]entity.Vector{entity.FloatVector(vector)}, // queryVectors
		"embedding", // vectorFieldName
		entity.L2,   // metricType,
		topK,        // topK
		&searchParam, // entity.SearchParam
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search embeddings in Milvus: %w", err)
	}

	return res, nil
}

// GetEmbedding retrieves an embedding from Milvus by repository ID.
func (mc *MilvusConnection) GetEmbedding(repoID int64) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	expr := fmt.Sprintf("repo_id == %d", repoID)
	outputFields := []string{"embedding"}

	resultSet, err := mc.Client.Query(ctx, mc.CollectionName, nil, expr, outputFields)
	if err != nil {
		return nil, fmt.Errorf("failed to query embedding from Milvus: %w", err)
	}

	if len(resultSet) == 0 || resultSet[0].Len() == 0 {
		return nil, fmt.Errorf("embedding not found for repo_id %d", repoID)
	}

	// Assuming 'embedding' is a float vector field
	var embeddingCol entity.Column
	for _, col := range resultSet {
		if col.Name() == "embedding" {
			embeddingCol = col
			break
		}
	}

	if embeddingCol == nil {
		return nil, fmt.Errorf("embedding column not found in Milvus query result")
	}

	// Extract the float vector
	vec, ok := embeddingCol.(*entity.ColumnFloatVector)
	if !ok || len(vec.Data()) == 0 {
		return nil, fmt.Errorf("invalid embedding format or empty embedding for repo_id %d", repoID)
	}

	return vec.Data()[0], nil
}

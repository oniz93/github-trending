package database

import (
	"context"
	"fmt"
	"log"

	qdrant_go_client "github.com/qdrant/go-client/qdrant" // Alias to avoid name collision
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type QdrantConnection struct {
	conn              *grpc.ClientConn
	pointsClient      qdrant_go_client.PointsClient
	collectionsClient qdrant_go_client.CollectionsClient
}

func NewQdrantConnection(host string, port int) (*QdrantConnection, error) {
	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", host, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}

	pointsClient := qdrant_go_client.NewPointsClient(conn)
	collectionsClient := qdrant_go_client.NewCollectionsClient(conn)

	return &QdrantConnection{
		conn:              conn,
		pointsClient:      pointsClient,
		collectionsClient: collectionsClient,
	}, nil
}

func (c *QdrantConnection) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *QdrantConnection) CreateCollection(ctx context.Context, collectionName string, vectorSize uint64) error {
	_, err := c.collectionsClient.Get(ctx, &qdrant_go_client.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})
	if err == nil {
		log.Printf("Collection %s already exists", collectionName)
		return nil
	}

	_, err = c.collectionsClient.Create(ctx, &qdrant_go_client.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant_go_client.VectorsConfig{
			Config: &qdrant_go_client.VectorsConfig_Params{
				Params: &qdrant_go_client.VectorParams{
					Size:     vectorSize,
					Distance: qdrant_go_client.Distance_Cosine,
				},
			},
		},
	})
	return err
}

func (c *QdrantConnection) UpsertVectors(ctx context.Context, collectionName string, points []*qdrant_go_client.PointStruct) error {
	// Add the id to the payload
	for _, point := range points {
		if point.Payload == nil {
			point.Payload = make(map[string]*qdrant_go_client.Value)
		}
		point.Payload["id"] = &qdrant_go_client.Value{
			Kind: &qdrant_go_client.Value_IntegerValue{
				IntegerValue: int64(point.GetId().GetNum()),
			},
		}
	}

	wait := true
	_, err := c.pointsClient.Upsert(ctx, &qdrant_go_client.UpsertPoints{
		CollectionName: collectionName,
		Wait:           &wait,
		Points:         points,
	})
	return err
}

func (c *QdrantConnection) Search(ctx context.Context, collectionName string, vector []float32, limit uint64, excludeID uint64) ([]*qdrant_go_client.ScoredPoint, error) {
	res, err := c.pointsClient.Search(ctx, &qdrant_go_client.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          limit,
		Filter: &qdrant_go_client.Filter{
			MustNot: []*qdrant_go_client.Condition{
				{
					ConditionOneOf: &qdrant_go_client.Condition_Field{
						Field: &qdrant_go_client.FieldCondition{
							Key: "id",
							Match: &qdrant_go_client.Match{
								MatchValue: &qdrant_go_client.Match_Integer{
									Integer: int64(excludeID),
								},
							},
						},
					},
				},
			},
		},
		WithVectors: &qdrant_go_client.WithVectorsSelector{
			SelectorOptions: &qdrant_go_client.WithVectorsSelector_Enable{
				Enable: true,
			},
		},
		WithPayload: &qdrant_go_client.WithPayloadSelector{
			SelectorOptions: &qdrant_go_client.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return res.GetResult(), nil
}

func (c *QdrantConnection) GetVectors(ctx context.Context, collectionName string, ids []uint64) ([]*qdrant_go_client.RetrievedPoint, error) {
	var pointIds []*qdrant_go_client.PointId
	for _, id := range ids {
		pointIds = append(pointIds, &qdrant_go_client.PointId{
			PointIdOptions: &qdrant_go_client.PointId_Num{Num: id},
		})
	}

	withVectors := true
	res, err := c.pointsClient.Get(ctx, &qdrant_go_client.GetPoints{
		CollectionName: collectionName,
		Ids:            pointIds,
		WithVectors: &qdrant_go_client.WithVectorsSelector{
			SelectorOptions: &qdrant_go_client.WithVectorsSelector_Enable{
				Enable: withVectors,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return res.GetResult(), nil
}
package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"telis-api-gateway/internal/domain"
	"telis-api-gateway/pb"
)

type legalEngineClient struct {
	client pb.LegalEngineServiceClient
}

func NewLegalEngineClient(targetURL string) (domain.LegalEngineClient, error) {
	conn, err := grpc.Dial(targetURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := pb.NewLegalEngineServiceClient(conn)
	return &legalEngineClient{client: client}, nil
}

func (c *legalEngineClient) GetDocumentClauses(ctx context.Context, documentID string) ([]domain.DocumentClause, error) {
	req := &pb.GetClausesRequest{DocumentId: documentID}
	
	// Add timeout context
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.GetDocumentClauses(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get clauses from legal engine: %w", err)
	}

	var result []domain.DocumentClause
	for _, clause := range resp.Clauses {
		id, _ := uuid.Parse(clause.Id)
		docID, _ := uuid.Parse(clause.DocumentId)
		createdAt, _ := time.Parse(time.RFC3339, clause.CreatedAt)

		result = append(result, domain.DocumentClause{
			ID:            id,
			DocumentID:    docID,
			ClauseType:    clause.ClauseType,
			ClauseText:    clause.ClauseText,
			RiskLevel:     clause.RiskLevel,
			RiskReasoning: clause.RiskReasoning,
			CreatedAt:     createdAt,
		})
	}

	return result, nil
}

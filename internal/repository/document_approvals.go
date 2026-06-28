package repository

import (
	"context"
	"telis-api-gateway/internal/domain"
)

func (r *documentRepository) CreateApprovalWorkflow(ctx context.Context, approval *domain.ApprovalWorkflow) error {
	query := `
		INSERT INTO gateway.approval_workflows (document_id, requester_id, approver_id, status, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		approval.DocumentID, approval.RequesterID, approval.ApproverID, approval.Status, approval.Notes,
	).Scan(&approval.ID, &approval.CreatedAt, &approval.UpdatedAt)
}

func (r *documentRepository) UpdateApprovalWorkflowStatus(ctx context.Context, id string, status string, notes string) error {
	query := `
		UPDATE gateway.approval_workflows 
		SET status = $1, notes = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, status, notes, id)
	return err
}

func (r *documentRepository) GetApprovalsByDocumentID(ctx context.Context, documentID string) ([]domain.ApprovalWorkflow, error) {
	query := `
		SELECT id, document_id, requester_id, approver_id, status, notes, created_at, updated_at
		FROM gateway.approval_workflows
		WHERE document_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []domain.ApprovalWorkflow
	for rows.Next() {
		var a domain.ApprovalWorkflow
		if err := rows.Scan(&a.ID, &a.DocumentID, &a.RequesterID, &a.ApproverID, &a.Status, &a.Notes, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		approvals = append(approvals, a)
	}
	return approvals, nil
}

package usecase

import (
	"context"

	"github.com/google/uuid"
	"telis-api-gateway/internal/domain"
)

func (u *documentUsecase) RequestApproval(ctx context.Context, documentID string, requesterID string, approverID string, notes string) (*domain.ApprovalWorkflow, error) {
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		return nil, err
	}
	reqUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, err
	}
	appUUID, err := uuid.Parse(approverID)
	if err != nil {
		return nil, err
	}

	approval := &domain.ApprovalWorkflow{
		DocumentID:  docUUID,
		RequesterID: reqUUID,
		ApproverID:  appUUID,
		Status:      "PENDING_REVIEW",
		Notes:       notes,
	}

	if err := u.repo.CreateApprovalWorkflow(ctx, approval); err != nil {
		return nil, err
	}

	// State machine approval: Update document status
	if err := u.repo.UpdateDocumentStatus(ctx, documentID, "PENDING_APPROVAL"); err != nil {
		return nil, err
	}

	return approval, nil
}

func (u *documentUsecase) ReviewApproval(ctx context.Context, documentID string, approvalID string, reviewerID string, status string, notes string) error {
	err := u.repo.UpdateApprovalWorkflowStatus(ctx, approvalID, status, notes)
	if err != nil {
		return err
	}

	// Update document status based on approval outcome
	if status == "APPROVED" || status == "REJECTED" {
		if err := u.repo.UpdateDocumentStatus(ctx, documentID, status); err != nil {
			return err
		}
	}
	return nil
}

func (u *documentUsecase) GetDocumentApprovals(ctx context.Context, documentID string) ([]domain.ApprovalWorkflow, error) {
	return u.repo.GetApprovalsByDocumentID(ctx, documentID)
}

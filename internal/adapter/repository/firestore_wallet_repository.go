package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"pasargamex/internal/domain/entity"
	"pasargamex/internal/domain/repository"
	"pasargamex/pkg/utils"
)

type firestoreWalletRepository struct {
	client *firestore.Client
}

func NewFirestoreWalletRepository(client *firestore.Client) repository.WalletRepository {
	return &firestoreWalletRepository{
		client: client,
	}
}

func (r *firestoreWalletRepository) CreateWallet(ctx context.Context, wallet *entity.Wallet) error {
	_, err := r.client.Collection("wallets").Doc(wallet.ID).Set(ctx, wallet)
	return err
}

func (r *firestoreWalletRepository) GetWalletByID(ctx context.Context, walletID string) (*entity.Wallet, error) {
	doc, err := r.client.Collection("wallets").Doc(walletID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var wallet entity.Wallet
	if err := doc.DataTo(&wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (r *firestoreWalletRepository) GetWalletByUserID(ctx context.Context, userID string) (*entity.Wallet, error) {
	query := r.client.Collection("wallets").Where("userId", "==", userID).Limit(1)
	iter := query.Documents(ctx)
	doc, err := iter.Next()
	if err != nil {
		return nil, err
	}

	var wallet entity.Wallet
	if err := doc.DataTo(&wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (r *firestoreWalletRepository) UpdateWallet(ctx context.Context, wallet *entity.Wallet) error {
	wallet.UpdatedAt = time.Now()
	_, err := r.client.Collection("wallets").Doc(wallet.ID).Set(ctx, wallet)
	return err
}

func (r *firestoreWalletRepository) UpdateWalletBalance(ctx context.Context, walletID string, amount float64) (*entity.Wallet, error) {
	err := r.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docRef := r.client.Collection("wallets").Doc(walletID)
		doc, err := tx.Get(docRef)
		if err != nil {
			return err
		}

		var wallet entity.Wallet
		if err := doc.DataTo(&wallet); err != nil {
			return err
		}

		wallet.Balance += amount
		wallet.UpdatedAt = time.Now()
		wallet.LastTxnAt = time.Now()

		if wallet.Balance < 0 {
			return fmt.Errorf("insufficient balance")
		}

		return tx.Set(docRef, wallet)
	})

	if err != nil {
		return nil, err
	}

	return r.GetWalletByID(ctx, walletID)
}

func (r *firestoreWalletRepository) GetWalletCount(ctx context.Context) (int, error) {
	iter := r.client.Collection("wallets").Documents(ctx)
	defer iter.Stop()
	
	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error counting wallets: %v", err)
			return 0, nil // Return 0 instead of error
		}
		count++
	}
	
	return count, nil
}

func (r *firestoreWalletRepository) GetTotalBalance(ctx context.Context) (float64, error) {
	iter := r.client.Collection("wallets").Documents(ctx)
	defer iter.Stop()
	
	totalBalance := 0.0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error calculating total balance: %v", err)
			return 0.0, nil // Return 0 instead of error
		}
		
		var wallet entity.Wallet
		if err := doc.DataTo(&wallet); err != nil {
			log.Printf("Error converting wallet document: %v", err)
			continue
		}
		
		totalBalance += wallet.Balance
	}
	
	return totalBalance, nil
}

type firestoreWalletTransactionRepository struct {
	client *firestore.Client
}

func NewFirestoreWalletTransactionRepository(client *firestore.Client) repository.WalletTransactionRepository {
	return &firestoreWalletTransactionRepository{
		client: client,
	}
}

func (r *firestoreWalletTransactionRepository) CreateTransaction(ctx context.Context, transaction *entity.WalletTransaction) error {
	_, err := r.client.Collection("wallet_transactions").Doc(transaction.ID).Set(ctx, transaction)
	return err
}

func (r *firestoreWalletTransactionRepository) GetTransactionByID(ctx context.Context, transactionID string) (*entity.WalletTransaction, error) {
	doc, err := r.client.Collection("wallet_transactions").Doc(transactionID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var transaction entity.WalletTransaction
	if err := doc.DataTo(&transaction); err != nil {
		return nil, err
	}

	return &transaction, nil
}

func (r *firestoreWalletTransactionRepository) GetTransactionsByWalletID(ctx context.Context, walletID string, pagination *utils.Pagination) ([]entity.WalletTransaction, error) {
	query := r.client.Collection("wallet_transactions").Where("walletId", "==", walletID).OrderBy("createdAt", firestore.Desc)
	
	if pagination.Page > 1 {
		query = query.Offset((pagination.Page - 1) * pagination.Limit)
	}
	query = query.Limit(pagination.Limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var transactions []entity.WalletTransaction
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var transaction entity.WalletTransaction
		if err := doc.DataTo(&transaction); err != nil {
			log.Printf("Error converting document to transaction: %v", err)
			continue
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (r *firestoreWalletTransactionRepository) GetTransactionsByUserID(ctx context.Context, userID string, pagination *utils.Pagination) ([]entity.WalletTransaction, error) {
	// Simple query without OrderBy to avoid composite index requirement
	query := r.client.Collection("wallet_transactions").Where("userId", "==", userID)
	
	if pagination.Page > 1 {
		query = query.Offset((pagination.Page - 1) * pagination.Limit)
	}
	query = query.Limit(pagination.Limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var transactions []entity.WalletTransaction
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating transactions: %v", err)
			// Return empty slice consistently
			return []entity.WalletTransaction{}, nil
		}

		var transaction entity.WalletTransaction
		if err := doc.DataTo(&transaction); err != nil {
			log.Printf("Error converting document to transaction: %v", err)
			continue
		}

		transactions = append(transactions, transaction)
	}

	// Always return non-nil slice
	if transactions == nil {
		transactions = []entity.WalletTransaction{}
	}

	return transactions, nil
}

func (r *firestoreWalletTransactionRepository) UpdateTransaction(ctx context.Context, transaction *entity.WalletTransaction) error {
	transaction.UpdatedAt = time.Now()
	_, err := r.client.Collection("wallet_transactions").Doc(transaction.ID).Set(ctx, transaction)
	return err
}

func (r *firestoreWalletTransactionRepository) GetTransactionsByType(ctx context.Context, userID string, txnType string, pagination *utils.Pagination) ([]entity.WalletTransaction, error) {
	query := r.client.Collection("wallet_transactions").Where("userId", "==", userID).Where("type", "==", txnType).OrderBy("createdAt", firestore.Desc)
	
	if pagination.Page > 1 {
		query = query.Offset((pagination.Page - 1) * pagination.Limit)
	}
	query = query.Limit(pagination.Limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var transactions []entity.WalletTransaction
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var transaction entity.WalletTransaction
		if err := doc.DataTo(&transaction); err != nil {
			log.Printf("Error converting document to transaction: %v", err)
			continue
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (r *firestoreWalletTransactionRepository) GetDailyTransactionCount(ctx context.Context) (int, error) {
	// Get transactions from last 24 hours
	yesterday := time.Now().AddDate(0, 0, -1)
	
	iter := r.client.Collection("wallet_transactions").Where("createdAt", ">=", yesterday).Documents(ctx)
	defer iter.Stop()
	
	count := 0
	for {
		_, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error counting daily transactions: %v", err)
			return 0, nil
		}
		count++
	}
	
	return count, nil
}

func (r *firestoreWalletTransactionRepository) GetDailyTransactionVolume(ctx context.Context) (float64, error) {
	// Get transactions from last 24 hours
	yesterday := time.Now().AddDate(0, 0, -1)
	
	iter := r.client.Collection("wallet_transactions").Where("createdAt", ">=", yesterday).Documents(ctx)
	defer iter.Stop()
	
	volume := 0.0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error calculating daily transaction volume: %v", err)
			return 0.0, nil
		}
		
		var transaction entity.WalletTransaction
		if err := doc.DataTo(&transaction); err != nil {
			log.Printf("Error converting transaction document: %v", err)
			continue
		}
		
		// Add absolute value of amount to get total volume
		if transaction.Amount < 0 {
			volume += -transaction.Amount
		} else {
			volume += transaction.Amount
		}
	}
	
	return volume, nil
}

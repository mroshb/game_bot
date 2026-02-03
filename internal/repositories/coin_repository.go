package repositories

import (
	"fmt"

	"github.com/mroshb/game_bot/internal/models"
	"github.com/mroshb/game_bot/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CoinRepository struct {
	db *gorm.DB
}

func NewCoinRepository(db *gorm.DB) *CoinRepository {
	return &CoinRepository{db: db}
}

// DeductCoins deducts coins from user's balance with transaction logging
func (r *CoinRepository) DeductCoins(userID uint, amount int64, txType, description string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get current balance
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New(errors.ErrCodeNotFound, "user not found")
			}
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get user")
		}

		// Check sufficient balance
		if user.CoinBalance < amount {
			return errors.New(errors.ErrCodeInsufficientFunds, fmt.Sprintf("insufficient coins: have %d, need %d", user.CoinBalance, amount))
		}

		// Update balance
		newBalance := user.CoinBalance - amount
		if err := tx.Model(&user).Update("coin_balance", newBalance).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to update balance")
		}

		// Create transaction record (negative amount for deduction)
		transaction := &models.CoinTransaction{
			UserID:          userID,
			Amount:          -amount,
			TransactionType: txType,
			Description:     description,
		}
		if err := tx.Create(transaction).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to create transaction")
		}

		return nil
	})
}

// AddCoins adds coins to user's balance with transaction logging
func (r *CoinRepository) AddCoins(userID uint, amount int64, txType, description string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get current balance
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New(errors.ErrCodeNotFound, "user not found")
			}
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to get user")
		}

		// Update balance
		newBalance := user.CoinBalance + amount
		if err := tx.Model(&user).Update("coin_balance", newBalance).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to update balance")
		}

		// Create transaction record (positive amount for addition)
		transaction := &models.CoinTransaction{
			UserID:          userID,
			Amount:          amount,
			TransactionType: txType,
			Description:     description,
		}
		if err := tx.Create(transaction).Error; err != nil {
			return errors.Wrap(err, errors.ErrCodeInternalError, "failed to create transaction")
		}

		return nil
	})
}

// GetBalance retrieves user's current coin balance
func (r *CoinRepository) GetBalance(userID uint) (int64, error) {
	var user models.User
	result := r.db.Select("coin_balance").First(&user, userID)

	if result.Error == gorm.ErrRecordNotFound {
		return 0, errors.New(errors.ErrCodeNotFound, "user not found")
	}
	if result.Error != nil {
		return 0, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get balance")
	}

	return user.CoinBalance, nil
}

// GetTransactionHistory retrieves user's transaction history
func (r *CoinRepository) GetTransactionHistory(userID uint, limit int) ([]models.CoinTransaction, error) {
	var transactions []models.CoinTransaction
	result := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&transactions)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, errors.ErrCodeInternalError, "failed to get transaction history")
	}

	return transactions, nil
}

// HasSufficientBalance checks if user has enough coins
func (r *CoinRepository) HasSufficientBalance(userID uint, amount int64) (bool, error) {
	balance, err := r.GetBalance(userID)
	if err != nil {
		return false, err
	}
	return balance >= amount, nil
}

package service
import(
	"fmt"

	"gitlab.com/my-game873206/auth-service/internal/model"
	"gitlab.com/my-game873206/auth-service/internal/repository"

	"github.com/google/uuid"
)

type GameAccountService struct {
	repo *repository.GameAccountRepository
}

func NewGameAccountService(repo *repository.GameAccountRepository) *GameAccountService {
	return &GameAccountService{repo: repo}
}

func (s *GameAccountService) List(userID uuid.UUID) ([]model.GameAccount, error) {
	accounts, err := s.repo.ListByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("GameAccountService.List: %w", err)
	}
	return accounts, nil
}

func (s *GameAccountService) Create(userID uuid.UUID, uid string) (*model.GameAccount, error) {
	account, err := s.repo.Create(userID, uid)
	if err != nil {
		return nil, fmt.Errorf("GameAccountService.Create: %w", err)
	}
	return account, nil
}

func (s *GameAccountService) Delete(id uuid.UUID, uid string) error {
	err := s.repo.Delete(id, uid)
	if err != nil {
		return fmt.Errorf("GameAccountService.Delete: %w", err)
	}
	return nil
}
package chat

import (
	"context"
	"errors"

	"workplace/internal/user"
)

type Service struct {
	repo     *Repository
	userRepo *user.Repository
}

func NewService(repo *Repository, userRepo *user.Repository) *Service {
	return &Service{repo: repo, userRepo: userRepo}
}

func (s *Service) CreateRoom(ctx context.Context, dto CreateRoomDTO, createdBy, creatorRole string) (*Room, error) {
	if dto.Name == "" {
		return nil, errors.New("room name is required")
	}
	if dto.Type == "" {
		dto.Type = RoomGeneral
	}
	room := &Room{
		Name:         dto.Name,
		Type:         dto.Type,
		DepartmentID: dto.DepartmentID,
		ProjectID:    dto.ProjectID,
		CreatedBy:    createdBy,
	}
	if err := s.repo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}
	_ = s.repo.AddMember(ctx, room.ID, createdBy)

	// Super admin creating a general room — add every user
	if creatorRole == "super_admin" && dto.Type == RoomGeneral {
		if all, err := s.userRepo.List(ctx); err == nil {
			for _, u := range all {
				_ = s.repo.AddMember(ctx, room.ID, u.ID)
			}
		}
		return room, nil
	}

	// Dept head creating any room — add all dept members
	if dto.DepartmentID != nil && *dto.DepartmentID != "" {
		if members, err := s.userRepo.ListByDepartment(ctx, *dto.DepartmentID); err == nil {
			for _, m := range members {
				_ = s.repo.AddMember(ctx, room.ID, m.ID)
			}
		}
	}

	return room, nil
}

func (s *Service) CreateDM(ctx context.Context, dto CreateDMDTO, currentUserID string) (*Room, error) {
	if dto.UserID == "" {
		return nil, errors.New("user_id is required")
	}
	if dto.UserID == currentUserID {
		return nil, errors.New("cannot create DM with yourself")
	}
	return s.repo.FindOrCreateDMRoom(ctx, currentUserID, dto.UserID)
}

func (s *Service) CreateProjectRoom(ctx context.Context, projectID, projectTitle, createdBy string) (*Room, error) {
	room := &Room{
		Name:      projectTitle + " — Project Chat",
		Type:      RoomProject,
		ProjectID: &projectID,
		CreatedBy: createdBy,
	}
	if err := s.repo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}
	_ = s.repo.AddMember(ctx, room.ID, createdBy)
	return room, nil
}

func (s *Service) GetProjectRoom(ctx context.Context, projectID string) (*Room, error) {
	return s.repo.FindRoomByProjectID(ctx, projectID)
}

func (s *Service) SyncProjectMember(ctx context.Context, projectID, userID string, add bool) error {
	room, err := s.repo.FindRoomByProjectID(ctx, projectID)
	if err != nil {
		return nil
	}
	if add {
		return s.repo.AddMember(ctx, room.ID, userID)
	}
	return s.repo.RemoveMember(ctx, room.ID, userID)
}

func (s *Service) ListRooms(ctx context.Context, userID string) ([]*Room, error) {
	return s.repo.ListRooms(ctx, userID)
}

func (s *Service) GetMessages(ctx context.Context, roomID, userID string) ([]*Message, error) {
	ok, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("you are not a member of this room")
	}
	return s.repo.ListMessages(ctx, roomID, 50)
}

func (s *Service) MarkRead(ctx context.Context, roomID, userID string) error {
	ok, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil || !ok {
		return errors.New("you are not a member of this room")
	}
	return s.repo.MarkRead(ctx, roomID, userID)
}

func (s *Service) DeleteMessage(ctx context.Context, messageID, userID string) error {
	return s.repo.DeleteMessage(ctx, messageID, userID)
}

func (s *Service) AddReaction(ctx context.Context, messageID, userID, emoji string) error {
	if emoji == "" {
		return errors.New("emoji is required")
	}
	return s.repo.AddReaction(ctx, messageID, userID, emoji)
}

func (s *Service) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	return s.repo.RemoveReaction(ctx, messageID, userID, emoji)
}

func (s *Service) ListReactions(ctx context.Context, messageID string) ([]*MessageReaction, error) {
	return s.repo.ListReactions(ctx, messageID)
}

func (s *Service) DeleteRoom(ctx context.Context, roomID, requestorID, requestorRole string) error {
	room, err := s.repo.FindRoomByID(ctx, roomID)
	if err != nil {
		return errors.New("room not found")
	}
	if requestorRole != "super_admin" && room.CreatedBy != requestorID {
		return errors.New("only the room creator or super admin can delete this room")
	}
	return s.repo.DeleteRoom(ctx, roomID)
}

func (s *Service) AddMember(ctx context.Context, roomID, userID string) error {
	return s.repo.AddMember(ctx, roomID, userID)
}

func (s *Service) RemoveMember(ctx context.Context, roomID, userID string) error {
	return s.repo.RemoveMember(ctx, roomID, userID)
}

func (s *Service) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	return s.repo.IsMember(ctx, roomID, userID)
}

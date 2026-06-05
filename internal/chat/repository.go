package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateRoom(ctx context.Context, room *Room) error {
	q := `INSERT INTO chat_rooms (name, type, department_id, project_id, created_by)
	      VALUES ($1, $2, $3, $4, $5)
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q,
		room.Name, room.Type, room.DepartmentID, room.ProjectID, room.CreatedBy,
	).Scan(&room.ID, &room.CreatedAt)
}

func (r *Repository) FindRoomByID(ctx context.Context, id string) (*Room, error) {
	q := `SELECT id, name, type, department_id, project_id, created_by, created_at
	      FROM chat_rooms WHERE id = $1`
	room := &Room{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&room.ID, &room.Name, &room.Type,
		&room.DepartmentID, &room.ProjectID, &room.CreatedBy, &room.CreatedAt,
	)
	return room, err
}

func (r *Repository) FindRoomByProjectID(ctx context.Context, projectID string) (*Room, error) {
	q := `SELECT id, name, type, department_id, project_id, created_by, created_at
	      FROM chat_rooms WHERE project_id = $1 AND type = 'project'`
	room := &Room{}
	err := r.db.QueryRow(ctx, q, projectID).Scan(
		&room.ID, &room.Name, &room.Type,
		&room.DepartmentID, &room.ProjectID, &room.CreatedBy, &room.CreatedAt,
	)
	return room, err
}

// FindOrCreateDMRoom finds an existing direct room between two users or creates one.
func (r *Repository) FindOrCreateDMRoom(ctx context.Context, userA, userB string) (*Room, error) {
	q := `SELECT cr.id, cr.name, cr.type, cr.department_id, cr.project_id, cr.created_by, cr.created_at
	      FROM chat_rooms cr
	      INNER JOIN chat_room_members m1 ON m1.room_id = cr.id AND m1.user_id = $1
	      INNER JOIN chat_room_members m2 ON m2.room_id = cr.id AND m2.user_id = $2
	      WHERE cr.type = 'direct'
	      LIMIT 1`
	room := &Room{}
	err := r.db.QueryRow(ctx, q, userA, userB).Scan(
		&room.ID, &room.Name, &room.Type,
		&room.DepartmentID, &room.ProjectID, &room.CreatedBy, &room.CreatedAt,
	)
	if err == nil {
		return room, nil
	}

	// Create a new DM room
	room = &Room{
		Name:      fmt.Sprintf("dm:%s:%s", userA, userB),
		Type:      RoomDirect,
		CreatedBy: userA,
	}
	if err := r.CreateRoom(ctx, room); err != nil {
		return nil, err
	}
	if err := r.AddMember(ctx, room.ID, userA); err != nil {
		return nil, err
	}
	if err := r.AddMember(ctx, room.ID, userB); err != nil {
		return nil, err
	}
	return room, nil
}

func (r *Repository) ListRooms(ctx context.Context, userID string) ([]*Room, error) {
	q := `SELECT cr.id, cr.name, cr.type, cr.department_id, cr.project_id, cr.created_by, cr.created_at
	      FROM chat_rooms cr
	      INNER JOIN chat_room_members crm ON crm.room_id = cr.id
	      WHERE crm.user_id = $1
	      ORDER BY cr.created_at DESC`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*Room
	for rows.Next() {
		room := &Room{}
		if err := rows.Scan(&room.ID, &room.Name, &room.Type,
			&room.DepartmentID, &room.ProjectID, &room.CreatedBy, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (r *Repository) DeleteRoom(ctx context.Context, roomID string) error {
	// Delete related rows first, then the room itself
	r.db.Exec(ctx, `DELETE FROM message_reactions WHERE message_id IN (SELECT id FROM messages WHERE room_id=$1)`, roomID)
	r.db.Exec(ctx, `DELETE FROM messages WHERE room_id=$1`, roomID)
	r.db.Exec(ctx, `DELETE FROM chat_room_members WHERE room_id=$1`, roomID)
	_, err := r.db.Exec(ctx, `DELETE FROM chat_rooms WHERE id=$1`, roomID)
	return err
}

func (r *Repository) AddMember(ctx context.Context, roomID, userID string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO chat_room_members (room_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		roomID, userID,
	)
	return err
}

func (r *Repository) RemoveMember(ctx context.Context, roomID, userID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM chat_room_members WHERE room_id=$1 AND user_id=$2`,
		roomID, userID,
	)
	return err
}

func (r *Repository) IsMember(ctx context.Context, roomID, userID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM chat_room_members WHERE room_id=$1 AND user_id=$2`,
		roomID, userID,
	).Scan(&count)
	return count > 0, err
}

func (r *Repository) GetRoomMemberIDs(ctx context.Context, roomID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id FROM chat_room_members WHERE room_id = $1`, roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *Repository) MarkRead(ctx context.Context, roomID, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE chat_room_members SET last_read_at=$1 WHERE room_id=$2 AND user_id=$3`,
		time.Now(), roomID, userID,
	)
	return err
}

func (r *Repository) SaveMessage(ctx context.Context, msg *Message) error {
	q := `INSERT INTO messages (room_id, sender_id, content, reply_to_id)
	      VALUES ($1, $2, $3, $4)
	      RETURNING id, created_at`
	return r.db.QueryRow(ctx, q, msg.RoomID, msg.SenderID, msg.Content, msg.ReplyToID).
		Scan(&msg.ID, &msg.CreatedAt)
}

func (r *Repository) DeleteMessage(ctx context.Context, id, senderID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE messages SET is_deleted=true, content='[deleted]' WHERE id=$1 AND sender_id=$2`,
		id, senderID,
	)
	return err
}

func (r *Repository) ListMessages(ctx context.Context, roomID string, limit int) ([]*Message, error) {
	if limit == 0 {
		limit = 50
	}
	q := `SELECT id, room_id, sender_id, content, reply_to_id, is_deleted, created_at
	      FROM messages WHERE room_id = $1
	      ORDER BY created_at DESC LIMIT $2`
	rows, err := r.db.Query(ctx, q, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.SenderID,
			&msg.Content, &msg.ReplyToID, &msg.IsDeleted, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (r *Repository) AddReaction(ctx context.Context, messageID, userID, emoji string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO message_reactions (message_id, user_id, emoji) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		messageID, userID, emoji,
	)
	return err
}

func (r *Repository) RemoveReaction(ctx context.Context, messageID, userID, emoji string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM message_reactions WHERE message_id=$1 AND user_id=$2 AND emoji=$3`,
		messageID, userID, emoji,
	)
	return err
}

func (r *Repository) ListReactions(ctx context.Context, messageID string) ([]*MessageReaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT message_id, user_id, emoji, created_at FROM message_reactions WHERE message_id=$1`,
		messageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*MessageReaction
	for rows.Next() {
		mr := &MessageReaction{}
		if err := rows.Scan(&mr.MessageID, &mr.UserID, &mr.Emoji, &mr.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, mr)
	}
	return list, nil
}

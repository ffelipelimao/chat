package db

import "chat-poc/entity"

func SaveMessage(username, content string) error {
	stmt, err := db.Prepare("INSERT INTO messages(username, content) VALUES(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(username, content)
	return err
}

func GetMessages(limit int) ([]entity.Message, error) {
	rows, err := db.Query("SELECT id, username, content, timestamp FROM messages ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []entity.Message
	for rows.Next() {
		var msg entity.Message
		var id int
		if err := rows.Scan(&id, &msg.Username, &msg.Content, &msg.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

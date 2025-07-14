package db

import "chat-poc/entity"

func SaveUserActivity(username, eventType string) error {
	stmt, err := db.Prepare("INSERT INTO user_activity(username, event_type) VALUES(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(username, eventType)
	return err
}

func GetUserActivities(limit int) ([]entity.UserActivity, error) {
	rows, err := db.Query("SELECT id, username, event_type, timestamp FROM user_activity ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []entity.UserActivity
	for rows.Next() {
		var activity entity.UserActivity
		if err := rows.Scan(&activity.ID, &activity.Username, &activity.EventType, &activity.Timestamp); err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}
	return activities, nil
}

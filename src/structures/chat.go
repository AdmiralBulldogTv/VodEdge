package structures

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Chat struct {
	ID    primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	VodID primitive.ObjectID `json:"vod_id" bson:"vod_id"`

	Twitch ChatTwitch `json:"twitch" bson:"twitch"`

	Timestamp time.Time `json:"timestamp" bson:"timestamp"`

	Content string `json:"content" bson:"content"`

	Badges []ChatBadge `json:"badges" bson:"badges"`
	Emotes []ChatEmote `json:"emotes" bson:"chat_emote"`
}

type ChatTwitch struct {
	ID          string `json:"id" bson:"id"`
	UserID      string `json:"user_id" bson:"user_id"`
	Login       string `json:"login" bson:"login"`
	DisplayName string `json:"display_name" bson:"display_name"`
	Color       string `json:"color" bson:"color"`
}

type ChatBadge struct {
	Name string   `json:"name" bson:"name"`
	URLs []string `json:"urls" bson:"urls"`
}

type ChatEmote struct {
	Name string   `json:"name" bson:"name"`
	URLs []string `json:"urls" bson:"urls"`
}

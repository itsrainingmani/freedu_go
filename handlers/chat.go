package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/willybeans/freedu_go/database"
	"github.com/willybeans/freedu_go/queries"
	"github.com/willybeans/freedu_go/types"
)

type Chat struct {
	ID          string `json:"id"`
	TimeCreated string `json:"time_created"`
	ChatName    string `json:"chat_name"`
}

type NewChat struct {
	ChatName string   `json:"chat_name"`
	Members  []string `json:"members"`
}

type ChatXref struct {
	ChatName string   `json:"chat_name"`
	Members  []string `json:"members"`
}

func GetMessagesByChatIDHandler(w http.ResponseWriter, r *http.Request) {
	chatId := r.URL.Query().Get("id")

	message, err := queries.GetMessagesByChatID(chatId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

func GetAllChatsForUserHandler(w http.ResponseWriter, r *http.Request) {
	userId := r.URL.Query().Get("id")

	GetChatRoomsByUserID, err := queries.GetChatRoomsByUserID(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetChatRoomsByUserID)
}

func GetAllXRefForChatHandler(w http.ResponseWriter, r *http.Request) {
	chatId := r.URL.Query().Get("id")

	GetXRefsByChatID, err := queries.GetXRefsByChatID(chatId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetXRefsByChatID)
}

func NewMessageForUserInChatHandler(w http.ResponseWriter, r *http.Request) {
	var newMessage types.NewMessage
	if err := json.NewDecoder(r.Body).Decode(&newMessage); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	message, err := queries.NewMessageForUserInChat(newMessage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

func NewChatHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Insert chat into chat table
	var newChat NewChat
	if err := json.NewDecoder(r.Body).Decode(&newChat); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var chat Chat
	err := database.DB().QueryRow("INSERT INTO chat_room (chat_name) VALUES ($1) RETURNING *", newChat.ChatName).Scan(&chat.ID, &chat.ChatName, &chat.TimeCreated)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Insert row for each member into user_chat_xref
	// first, create vals array with each uuid
	vals := []interface{}{}
	valStrings := make([]string, 0, len(newChat.Members))
	// i is an incrementer for postgres $1 $2 etc
	i := 0
	for _, memberId := range newChat.Members {
		valStrings = append(valStrings, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		// userId then chat ID for each insert
		vals = append(vals, memberId, chat.ID)
		i++
	}
	// then craft sql statement for inserting data into xref table
	sqlStr := fmt.Sprintf("INSERT INTO user_chatroom_xref (user_id, chat_room_id) VALUES %s", strings.Join(valStrings, ","))

	// now insert into database
	// potential improvement: return values to check for error
	rows, xrefErr := database.DB().Query(sqlStr, vals...)
	if xrefErr != nil {
		http.Error(w, xrefErr.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

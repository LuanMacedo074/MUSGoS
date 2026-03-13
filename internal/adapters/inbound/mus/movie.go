package mus

import (
	"fmt"
	"fsos-server/internal/domain/ports"
	"sync"
)

type Movie struct {
	Name   string
	groups map[string]*Group
	mu     sync.RWMutex
}

func newMovie(name string) *Movie {
	return &Movie{
		Name:   name,
		groups: make(map[string]*Group),
	}
}

func (m *Movie) AddGroup(name string, group *Group) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groups[name] = group
}

func (m *Movie) RemoveGroup(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.groups, name)
}

func (m *Movie) GetGroup(name string) (*Group, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.groups[name]
	return g, ok
}

func (m *Movie) GetGroupNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.groups))
	for name := range m.groups {
		names = append(names, name)
	}
	return names
}

func (m *Movie) GetGroupCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.groups)
}

type MovieManager struct {
	sessionStore ports.SessionStore
	logger       ports.Logger
	mu           sync.RWMutex
	movies       map[string]*Movie
}

func NewMovieManager(sessionStore ports.SessionStore, logger ports.Logger) *MovieManager {
	return &MovieManager{
		sessionStore: sessionStore,
		logger:       logger,
		movies:       make(map[string]*Movie),
	}
}

func movieRoomName(movieID string) string {
	return fmt.Sprintf("movie:%s", movieID)
}

func (mm *MovieManager) JoinMovie(movieID, userID string) error {
	mm.mu.Lock()
	movie, exists := mm.movies[movieID]
	if !exists {
		movie = newMovie(movieID)
		mm.movies[movieID] = movie
		mm.logger.Info("Movie created", map[string]interface{}{
			"movieID": movieID,
		})
	}
	mm.mu.Unlock()

	if err := mm.sessionStore.JoinRoom(movieRoomName(movieID), userID); err != nil {
		return fmt.Errorf("failed to join movie room: %w", err)
	}

	// Auto-create @AllUsers group if it doesn't exist.
	// Benign race: two concurrent JoinMovie calls on a new movie may both
	// call AddGroup("@AllUsers"). This is harmless because AddGroup overwrites
	// with an identical group (idempotent).
	if _, ok := movie.GetGroup("@AllUsers"); !ok {
		allUsers := NewGroup("@AllUsers", movieID, true)
		movie.AddGroup("@AllUsers", allUsers)
	}

	// Auto-join user to @AllUsers
	roomName := groupRoomName(movieID, "@AllUsers")
	if err := mm.sessionStore.JoinRoom(roomName, userID); err != nil {
		return fmt.Errorf("failed to join @AllUsers group: %w", err)
	}

	mm.logger.Info("User joined movie", map[string]interface{}{
		"movieID": movieID,
		"userID":  userID,
	})

	return nil
}

func (mm *MovieManager) LeaveMovie(movieID, userID string) error {
	mm.mu.Lock()
	movie, exists := mm.movies[movieID]
	if !exists {
		mm.mu.Unlock()
		return fmt.Errorf("movie %q not found", movieID)
	}

	// Leave all groups in this movie
	for _, groupName := range movie.GetGroupNames() {
		roomName := groupRoomName(movieID, groupName)
		mm.sessionStore.LeaveRoom(roomName, userID)
	}

	// Leave the movie room
	if err := mm.sessionStore.LeaveRoom(movieRoomName(movieID), userID); err != nil {
		mm.mu.Unlock()
		return fmt.Errorf("failed to leave movie room: %w", err)
	}

	// Destroy movie if empty
	members, err := mm.sessionStore.GetRoomMembers(movieRoomName(movieID))
	if err != nil {
		mm.mu.Unlock()
		return fmt.Errorf("failed to get movie members: %w", err)
	}

	if len(members) == 0 {
		delete(mm.movies, movieID)
		mm.mu.Unlock()
		mm.logger.Info("Movie destroyed (empty)", map[string]interface{}{
			"movieID": movieID,
		})
	} else {
		mm.mu.Unlock()
	}

	mm.logger.Info("User left movie", map[string]interface{}{
		"movieID": movieID,
		"userID":  userID,
	})

	return nil
}

func (mm *MovieManager) GetMovie(movieID string) (*Movie, bool) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	m, ok := mm.movies[movieID]
	return m, ok
}

func (mm *MovieManager) GetMovies() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	names := make([]string, 0, len(mm.movies))
	for name := range mm.movies {
		names = append(names, name)
	}
	return names
}

func (mm *MovieManager) GetMovieUsers(movieID string) ([]string, error) {
	return mm.sessionStore.GetRoomMembers(movieRoomName(movieID))
}

func (mm *MovieManager) GetMovieUserCount(movieID string) (int, error) {
	members, err := mm.sessionStore.GetRoomMembers(movieRoomName(movieID))
	if err != nil {
		return 0, err
	}
	return len(members), nil
}

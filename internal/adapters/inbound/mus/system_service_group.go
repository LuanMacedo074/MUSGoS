package mus

import (
	"fsos-server/internal/domain/types/lingo"
	"fsos-server/internal/domain/types/smus"
)

func (s *SystemService) handleGroupJoin(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	groupName, err := lingo.ExtractString(msg.MsgContent)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}

	// Ensure the group exists in the movie
	movie, ok := s.movieManager.GetMovie(movieID)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	if _, exists := movie.GetGroup(groupName); !exists {
		movie.AddGroup(groupName, NewGroup(groupName, movieID, false))
	}

	if err := s.groupManager.JoinGroup(movieID, groupName, senderID); err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLVoid()), nil
}

func (s *SystemService) handleGroupLeave(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	groupName, err := lingo.ExtractString(msg.MsgContent)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	if err := s.groupManager.LeaveGroup(movieID, groupName, senderID); err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLVoid()), nil
}

func (s *SystemService) handleGroupGetUsers(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	groupName, err := lingo.ExtractString(msg.MsgContent)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	members, err := s.groupManager.GetGroupMembers(movieID, groupName)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	list := lingo.NewLList()
	for _, m := range members {
		list.Values = append(list.Values, lingo.NewLString(m))
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, list), nil
}

func (s *SystemService) handleGroupGetUserCount(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	groupName, err := lingo.ExtractString(msg.MsgContent)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	count, err := s.groupManager.GetGroupMemberCount(movieID, groupName)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLInteger(int32(count))), nil
}

func (s *SystemService) handleGroupSetAttribute(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	plist, ok := msg.MsgContent.(*lingo.LPropList)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	groupVal, err := plist.GetElement("group")
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	attrVal, err := plist.GetElement("attribute")
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	valueVal, err := plist.GetElement("value")
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}

	groupName := lingo.StringValue(groupVal)
	attrName := lingo.StringValue(attrVal)

	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	movie, ok := s.movieManager.GetMovie(movieID)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	group, ok := movie.GetGroup(groupName)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	group.SetAttribute(attrName, valueVal)
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLVoid()), nil
}

func (s *SystemService) handleGroupGetAttribute(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	plist, ok := msg.MsgContent.(*lingo.LPropList)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	groupVal, err := plist.GetElement("group")
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	attrVal, err := plist.GetElement("attribute")
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}

	groupName := lingo.StringValue(groupVal)
	attrName := lingo.StringValue(attrVal)

	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	movie, ok := s.movieManager.GetMovie(movieID)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	group, ok := movie.GetGroup(groupName)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	value := group.GetAttribute(attrName)
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, value), nil
}

func (s *SystemService) handleGroupDeleteAttribute(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	plist, ok := msg.MsgContent.(*lingo.LPropList)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	groupVal, err := plist.GetElement("group")
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	attrVal, err := plist.GetElement("attribute")
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}

	groupName := lingo.StringValue(groupVal)
	attrName := lingo.StringValue(attrVal)

	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	movie, ok := s.movieManager.GetMovie(movieID)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	group, ok := movie.GetGroup(groupName)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	group.DeleteAttribute(attrName)
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, lingo.NewLVoid()), nil
}

func (s *SystemService) handleGroupGetAttributeNames(senderID string, msg *smus.MUSMessage) (*smus.MUSMessage, error) {
	groupName, err := lingo.ExtractString(msg.MsgContent)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrInvalidMessageFormat, lingo.NewLVoid()), nil
	}
	movieID, err := s.getUserMovieID(senderID)
	if err != nil {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	movie, ok := s.movieManager.GetMovie(movieID)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	group, ok := movie.GetGroup(groupName)
	if !ok {
		return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrServerInternalError, lingo.NewLVoid()), nil
	}
	names := group.GetAttributeNames()
	list := lingo.NewLList()
	for _, name := range names {
		list.Values = append(list.Values, lingo.NewLString(name))
	}
	return NewResponse(msg.Subject.Value, "System", []string{senderID}, smus.ErrNoError, list), nil
}

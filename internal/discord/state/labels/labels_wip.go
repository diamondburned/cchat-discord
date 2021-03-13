package labels

// Unused things.

// // AddPresenceLabel adds a label to display the giveen presence live. Refer to
// // Repository for more information.
// func (r *Repository) AddPresenceLabel(uID discord.UserID, l cchat.LabelContainer) (func(), error) {
// 	presence, err := r.state.Presence(0, uID)
// 	if err != nil {
// 		// We can accept lazy presences.
// 		l.SetLabel(text.Plain(uID.Mention()))
// 	} else {
// 		l.SetLabel()
// 	}

// 	r.mutex.Lock()

// 	llist := r.stores.presences[uID]
// 	llist.Add(l)
// 	r.stores.presences[uID] = llist

// 	r.mutex.Unlock()

// 	return func() {
// 		r.mutex.Lock()
// 		defer r.mutex.Unlock()

// 		llist := r.stores.presences[uID]
// 		llist.Remove(l)

// 		if len(llist) == 0 {
// 			delete(r.stores.presences, uID)
// 			return
// 		}

// 		r.stores.presences[uID] = llist
// 	}
// }

// func newPresenceSegment(s *ningen.State, uID discord.UserID) (*mention.User, error) {
// 	p, err := s.Presence(0, uID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	user := mention.NewUser(p.User)
// 	user.WithState(s)
// 	user.WithMember(opItem.Member.Member)
// 	user.WithGuildID(ch.GuildID)
// 	user.WithPresence(opItem.Member.Presence)
// 	user.Prefetch()
// }

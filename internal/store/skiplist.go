package store

import "math/rand"

const (
	maxLevel    = 16
	probability = 0.25
)

type skipListNode struct {
	score  float64
	member string
	next   []*skipListNode
}

type skipList struct {
	head   *skipListNode
	levels int
	length int
	scores map[string]float64
}

func newSkipList() *skipList {
	return &skipList{
		head:   &skipListNode{next: make([]*skipListNode, maxLevel)},
		levels: 1,
		scores: make(map[string]float64),
	}
}

func (sl *skipList) randomLevel() int {
	level := 1
	for level < maxLevel && rand.Float64() < probability {
		level++
	}
	return level
}

// less reports whether (aScore, aMember) sorts before (bScore, bMember).
// Ties in score are broken lexicographically by member.
func (sl *skipList) less(aScore float64, aMember string, bScore float64, bMember string) bool {
	if aScore != bScore {
		return aScore < bScore
	}
	return aMember < bMember
}

// insert adds or updates a member. Returns true if the member is new.
func (sl *skipList) insert(score float64, member string) bool {
	isNew := true
	if oldScore, exists := sl.scores[member]; exists {
		isNew = false
		sl.remove(oldScore, member)
	}

	update := make([]*skipListNode, maxLevel)
	cur := sl.head
	for i := sl.levels - 1; i >= 0; i-- {
		for cur.next[i] != nil && sl.less(cur.next[i].score, cur.next[i].member, score, member) {
			cur = cur.next[i]
		}
		update[i] = cur
	}

	level := sl.randomLevel()
	if level > sl.levels {
		for i := sl.levels; i < level; i++ {
			update[i] = sl.head
		}
		sl.levels = level
	}

	node := &skipListNode{
		score:  score,
		member: member,
		next:   make([]*skipListNode, level),
	}
	for i := 0; i < level; i++ {
		node.next[i] = update[i].next[i]
		update[i].next[i] = node
	}

	sl.scores[member] = score
	sl.length++
	return isNew
}

// remove deletes a member by score+member.
func (sl *skipList) remove(score float64, member string) bool {
	update := make([]*skipListNode, maxLevel)
	cur := sl.head
	for i := sl.levels - 1; i >= 0; i-- {
		for cur.next[i] != nil && sl.less(cur.next[i].score, cur.next[i].member, score, member) {
			cur = cur.next[i]
		}
		update[i] = cur
	}

	target := update[0].next[0]
	if target == nil || target.member != member {
		return false
	}
	for i := 0; i < sl.levels; i++ {
		if update[i].next[i] != target {
			break
		}
		update[i].next[i] = target.next[i]
	}
	for sl.levels > 1 && sl.head.next[sl.levels-1] == nil {
		sl.levels--
	}
	delete(sl.scores, member)
	sl.length--
	return true
}

// score returns the score for a member. ok is false if the member does not exist.
func (sl *skipList) score(member string) (float64, bool) {
	s, ok := sl.scores[member]
	return s, ok
}

// rank returns the 0-based rank of a member by ascending score. Returns -1 if not found.
func (sl *skipList) rank(member string) int {
	r := 0
	cur := sl.head.next[0]
	for cur != nil {
		if cur.member == member {
			return r
		}
		r++
		cur = cur.next[0]
	}
	return -1
}

// rangeByRank returns members in [start, stop] (0-based, inclusive).
// Negative indices must be resolved by the caller before this call.
func (sl *skipList) rangeByRank(start, stop int) []ZSetMember {
	if start > stop || start >= sl.length {
		return []ZSetMember{}
	}
	if stop >= sl.length {
		stop = sl.length - 1
	}
	result := make([]ZSetMember, 0, stop-start+1)
	cur := sl.head.next[0]
	for i := 0; cur != nil; i++ {
		if i > stop {
			break
		}
		if i >= start {
			result = append(result, ZSetMember{Score: cur.score, Member: cur.member})
		}
		cur = cur.next[0]
	}
	return result
}

package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	// "fmt"
	// "log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labrpc"
)

const (
	FOLLOWER             int           = 0
	CANDIDATE            int           = 1
	LEADER               int           = 2
	HEARTSBEATS_INTERVAL time.Duration = time.Duration(150) * time.Millisecond
	TIMEINF              time.Duration = time.Duration(100000) * time.Hour
)

func max(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}

func min(a int, b int) int {
	if a <= b {
		return a
	}
	return b
}

func getRandTimeout() time.Duration {
	r := rand.New(rand.NewSource(time.Now().UnixMicro()))
	electionTimeOut := r.Int63()%(600-450) + 450 //随机产生的选举超时时间 450ms<= x <=600ms
	return time.Duration(electionTimeOut) * time.Millisecond
}

func (rf *Raft) switchState(state int) {
	if state == rf.state {
		if state == FOLLOWER {
			rf.votedFor = -1
		}
		return
	}
	rf.state = state
	switch state {
	case FOLLOWER:
		rf.votedFor = -1
		rf.heartBeatTimer.Stop()
		rf.electionTimer.Reset(getRandTimeout())
	case CANDIDATE:
		rf.heartBeatTimer.Stop()
	case LEADER:
		rf.heartBeatTimer.Reset(HEARTSBEATS_INTERVAL)
		rf.electionTimer.Stop()
		for peer := range rf.peers {
			rf.nextIndex[peer] = len(rf.log)
			rf.matchIndex[peer] = 0
		}
		// fmt.Println(len(rf.log))
	}
}

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

//log entry
type Entry struct {
	Command interface{}
	Term    int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	//persistent state on all servers
	currentTerm int     //服务器已知最新的任期（在服务器首次启动时初始化为0，单调递增）
	votedFor    int     //当前任期内收到选票的 candidateId，如果没有投给任何候选人 则为空
	log         []Entry //日志条目；每个条目包含了用于状态机的命令，以及领导人接收到该条目时的任期（初始索引为1）
	//volatile state on all servers
	commitIndex   int         //已知已提交的最高的日志条目的索引（初始值为0，单调递增）
	lastApplied   int         //已经被应用到状态机的最高的日志条目的索引（初始值为0，单调递增）
	state         int         //当前服务器的状态（follower,leader,candidate）
	electionTimer *time.Timer //选举超时定时器
	//volatile state on leaders
	heartBeatTimer *time.Timer //发送心跳的定时器
	nextIndex      []int       //对于每一台服务器，发送到该服务器的下一个日志条目的索引（初始值为领导人最后的日志条目的索引+1）
	matchIndex     []int       //对于每一台服务器，已知的已经复制到该服务器的最高日志条目的索引（初始值为0，单调递增）

	applyCh chan ApplyMsg
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var term int
	var isleader bool
	// // Your code here (2A).
	rf.mu.Lock()
	term = rf.currentTerm
	isleader = rf.state == LEADER
	rf.mu.Unlock()
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

/******************AppendEntries RPC*******************************/

//AppendEntries RPC arguments structure
type AppendEntriesArgs struct {
	Term         int     //领导人的任期
	LeaderId     int     //领导人 ID 因此跟随者可以对客户端进行重定向
	PreLogIndex  int     //紧邻新日志条目之前的那个日志条目的索引
	PreLogTerm   int     //紧邻新日志条目之前的那个日志条目的任期
	Entries      []Entry //需要被保存的日志条目（被当做心跳使用时，则日志条目内容为空；为了提高效率可能一次性发送多个）
	LeaderCommit int     //领导人的已知已提交的最高的日志条目的索引
}

//AppendEntries RPC reply structure
type AppendEntriesReply struct {
	Term    int  //当前任期，对于领导人而言 它会更新自己的任期
	Success bool //如果跟随者所含有的条目和 prevLogIndex 以及 prevLogTerm 匹配上了，则为 true
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	//1
	if rf.currentTerm > args.Term {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	rf.electionTimer.Reset(getRandTimeout())
	if rf.currentTerm < args.Term {
		rf.currentTerm = args.Term
		rf.switchState(FOLLOWER)
	}

	reply.Term = rf.currentTerm
	reply.Success = true
	//2	,3
	if len(rf.log)-1 < args.PreLogIndex || rf.log[args.PreLogIndex].Term != args.PreLogTerm {
		reply.Success = false
		return
	}
	//4
	rf.log = append(rf.log[:args.PreLogIndex+1], args.Entries...)
	// fmt.Println(rf.me, rf.log)

	//5
	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = min(args.LeaderCommit, len(rf.log)-1)
		rf.updateCommit()
	}

}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) sendHeatbeat() {
	// func (rf *Raft) leaderAppendEntries() {
	rf.mu.Lock()
	args := AppendEntriesArgs{Term: rf.currentTerm,
		LeaderId:     rf.me,
		PreLogIndex:  len(rf.log) - 1,
		PreLogTerm:   rf.log[len(rf.log)-1].Term,
		Entries:      make([]Entry, 0),
		LeaderCommit: rf.commitIndex,
	}
	rf.mu.Unlock()
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		go func(peer int) {
			reply := AppendEntriesReply{}
			ok := rf.sendAppendEntries(peer, &args, &reply)
			if !ok {
				// log.Printf("Peer[%d] can't receive leader[%d]'s heartbeat", peer, rf.me)
				return
			}
			// if reply.Success {
			// 	log.Printf("leader[%d] success transmit heartbeat to Peer[%d]", rf.me, peer)
			// }
			rf.mu.Lock()
			if !reply.Success && reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				rf.switchState(FOLLOWER)
				// log.Printf("find Term bigger then leader[%d],became follower", rf.me)
			}
			rf.mu.Unlock()
		}(peer)
	}
}

func (rf *Raft) leaderAppendEntries() {
	// func (rf *Raft) sendHeatbeat() {
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		go func(peer int) {
			rf.mu.Lock()
			args := AppendEntriesArgs{Term: rf.currentTerm,
				LeaderId:     rf.me,
				PreLogIndex:  rf.nextIndex[peer] - 1,
				PreLogTerm:   rf.log[rf.nextIndex[peer]-1].Term,
				LeaderCommit: rf.commitIndex,
			}

			args.Entries = make([]Entry, rf.nextIndex[rf.me]-rf.nextIndex[peer])
			copy(args.Entries, rf.log[rf.nextIndex[peer]:rf.nextIndex[rf.me]])

			if rf.nextIndex[peer] != rf.nextIndex[rf.me] {
				// log.Printf("start copy from leader[%d]", rf.me)
			}

			rf.mu.Unlock()
			reply := AppendEntriesReply{}
			ok := rf.sendAppendEntries(peer, &args, &reply)
			if !ok {
				// log.Printf("call peer[%d] failed", peer)
				return
			}
			rf.mu.Lock()
			defer rf.mu.Unlock()

			if reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				rf.switchState(FOLLOWER)
				// log.Printf("find Term bigger then Peer[%d],failed to become leader", rf.me)
				return
			}
			if reply.Success {
				if rf.nextIndex[peer] != rf.nextIndex[rf.me] {
					// log.Printf("Peer[%d] success copy log[%d:%d] from leader[%d]", peer, rf.nextIndex[peer], rf.nextIndex[rf.me]-1, rf.me)
				}
				rf.nextIndex[peer] = args.PreLogIndex + len(args.Entries) + 1
				rf.matchIndex[peer] = args.PreLogIndex + len(args.Entries)
				//假设存在 N 满足N > commitIndex ，
				//使得大多数的 matchIndex[i] ≥ N 以及log[N].term ==currentTerm 成立，则令 commitIndex = N
				for N := len(rf.log) - 1; N > rf.commitIndex; N-- {
					if rf.log[N].Term != rf.currentTerm {
						break
					}
					cnt := 0
					for peer := range rf.peers {
						if rf.matchIndex[peer] >= N {
							cnt++
						}
					}
					if 2*cnt >= len(rf.peers) {
						rf.commitIndex = N
						rf.updateCommit()
						break
					}
				}

			} else {
				rf.nextIndex[peer]--
			}
		}(peer)
	}
}

func (rf *Raft) updateCommit() {
	// rf.mu.Lock()
	// defer rf.mu.Unlock()
	if rf.commitIndex > rf.lastApplied {
		base := rf.lastApplied
		entries := rf.log[base+1 : rf.commitIndex+1]
		for idx, entry := range entries {
			msg := ApplyMsg{CommandValid: true,
				Command:      entry.Command,
				CommandIndex: base + idx + 1,
			}
			rf.applyCh <- msg
			rf.lastApplied = max(rf.lastApplied, msg.CommandIndex)
			// fmt.Printf("peer[%v] update commitIndex [%v]->[%v] logsLen:[%v] lastApplied[%v]\n", rf.me, rf.commitIndex, base+idx+1, len(rf.log), rf.lastApplied)

			// log.Printf("Peer[%d] success commit log[%d] to client", rf.me, base+idx+1)
		}

	}
	// fmt.Println(rf.me, rf.log, rf.commitIndex)

}

/******************RequestVote RPC*******************************/
//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int //候选人的任期号
	CandidateID  int //请求选票的候选人的ID
	LastLogIndex int //候选人的最后日志条目的索引值
	LastLogTerm  int //候选人最后日志条目的任期号
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int  //当前任期号，以便于候选人去更新自己的任期号
	VoteGranted bool //候选人赢得了此张选票时为真

}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.currentTerm
	reply.VoteGranted = false

	if rf.currentTerm > args.Term {
		return
	}

	if rf.currentTerm < args.Term {
		rf.currentTerm = args.Term
		rf.switchState(FOLLOWER)
	}

	if rf.votedFor == -1 || rf.votedFor == args.CandidateID {
		lastLogTerm := rf.log[len(rf.log)-1].Term
		if lastLogTerm < args.LastLogTerm || (lastLogTerm == args.LastLogTerm && len(rf.log)-1 <= args.LastLogIndex) {
			rf.switchState(FOLLOWER)
			rf.votedFor = args.CandidateID
			rf.electionTimer.Reset(getRandTimeout())
			reply.VoteGranted = true
		}
	}

}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) startElection() {
	rf.mu.Lock()
	rf.currentTerm++
	rf.votedFor = rf.me
	rf.electionTimer.Reset(getRandTimeout())
	args := RequestVoteArgs{Term: rf.currentTerm,
		CandidateID:  rf.me,
		LastLogIndex: len(rf.log) - 1,
		LastLogTerm:  rf.log[len(rf.log)-1].Term}
	rf.mu.Unlock()

	cond := sync.NewCond(&sync.Mutex{})
	cnt := 1      //接收到选票数
	finished := 1 //接收到总响应数

	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		go func(peer int) {
			reply := RequestVoteReply{}
			ok := rf.sendRequestVote(peer, &args, &reply)
			if !ok {
				// log.Printf("peer[%d] call peer[%d] failed", rf.me, peer)
				return
			}

			cond.L.Lock()
			if reply.VoteGranted {
				// log.Printf("Peer[%d] voted Peer[%d]", peer, rf.me)
				cnt++
			} else {
				rf.mu.Lock()
				if reply.Term > rf.currentTerm {
					rf.switchState(FOLLOWER)
					rf.currentTerm = reply.Term
					// log.Printf("find Term bigger then Peer[%d],failed to become leader", rf.me)
				}
				rf.mu.Unlock()
			}
			finished++
			cond.L.Unlock()
			cond.Broadcast()
		}(peer)
	}

	cond.L.Lock()
	defer cond.L.Unlock()
	for finished != len(rf.peers) && cnt*2 < len(rf.peers) {
		rf.mu.Lock()
		if rf.state != CANDIDATE {
			rf.mu.Unlock()
			break
		}
		rf.mu.Unlock()
		cond.Wait()
	}
	rf.mu.Lock()
	if cnt*2 >= len(rf.peers) && rf.state == CANDIDATE {
		rf.switchState(LEADER)
		// log.Printf("Peer[%d] become new leader", rf.me)
		rf.heartBeatTimer.Reset(10 * time.Millisecond)
	} else {
		// log.Printf("Peer[%d] failed to become leader", rf.me)
	}
	rf.mu.Unlock()

}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {

	// Your code here (2B).
	rf.mu.Lock()
	index := rf.nextIndex[rf.me]
	term := rf.currentTerm
	isLeader := rf.state == LEADER
	if isLeader {
		rf.log = append(rf.log, Entry{Command: command, Term: rf.currentTerm})
		rf.nextIndex[rf.me]++
		rf.matchIndex[rf.me] = len(rf.log) - 1
		rf.heartBeatTimer.Reset(time.Millisecond * 10)
		// log.Printf("add new log %d to leader[%d]'s log[%d]", command, rf.me, index)
	}
	rf.mu.Unlock()

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for !rf.killed() {
		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().

		select {
		case <-rf.heartBeatTimer.C: //leader
			rf.mu.Lock()
			go rf.leaderAppendEntries()
			rf.heartBeatTimer.Reset(HEARTSBEATS_INTERVAL)
			rf.mu.Unlock()

		case <-rf.electionTimer.C: //follower or candidate
			rf.mu.Lock()
			if rf.state == FOLLOWER {
				rf.switchState(CANDIDATE)
			}
			// log.Printf("Peer[%d] try to become leader", rf.me)
			go rf.startElection()
			rf.mu.Unlock()
		}
	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{
		peers:          peers,
		persister:      persister,
		me:             me,
		currentTerm:    0,
		votedFor:       -1,
		log:            make([]Entry, 0),
		commitIndex:    0,
		lastApplied:    0,
		nextIndex:      make([]int, len(peers)),
		matchIndex:     make([]int, len(peers)),
		state:          FOLLOWER,
		electionTimer:  time.NewTimer(getRandTimeout()),
		heartBeatTimer: time.NewTimer(TIMEINF),
		applyCh:        applyCh,
	}
	rf.log = append(rf.log, Entry{Term: 0})
	// Your initialization code here (2A, 2B, 2C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()
	return rf
}

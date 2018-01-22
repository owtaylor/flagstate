package util

import "sync"

type repoInfo struct {
	lowPriority  bool
	highPriority bool
	pending      bool
}

type repoDispatcher struct {
	repos             map[string]*repoInfo
	lowPriorityRepos  map[string]*repoInfo
	highPriorityRepos map[string]*repoInfo
	pendingCount      int

	mutex       sync.Mutex
	readyCond   *sync.Cond
	pendingCond *sync.Cond
	locked      bool
}

func NewRepoDispatcher() *repoDispatcher {
	rd := &repoDispatcher{
		repos:             make(map[string]*repoInfo),
		lowPriorityRepos:  make(map[string]*repoInfo),
		highPriorityRepos: make(map[string]*repoInfo),
	}
	rd.readyCond = sync.NewCond(&rd.mutex)
	rd.pendingCond = sync.NewCond(&rd.mutex)

	return rd
}

func (rd *repoDispatcher) Lock() {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	rd.locked = true
	for rd.pendingCount > 0 {
		rd.pendingCond.Wait()
	}
}

func (rd *repoDispatcher) Unlock() {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	rd.locked = false
	if rd.someReady() {
		rd.readyCond.Broadcast()
	}
}

func (rd *repoDispatcher) addToReady(repo string, info *repoInfo) {
	if info.lowPriority {
		rd.lowPriorityRepos[repo] = info
	} else if info.highPriority {
		delete(rd.lowPriorityRepos, repo)
		rd.highPriorityRepos[repo] = info
	}
}

func (rd *repoDispatcher) someReady() bool {
	return !rd.locked && (len(rd.highPriorityRepos) > 0 || len(rd.lowPriorityRepos) > 0)
}

func (rd *repoDispatcher) Add(repo string, lowPriority bool) {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	info := rd.repos[repo]
	if info == nil {
		info = &repoInfo{}
		rd.repos[repo] = info
	}

	added := !(info.highPriority || info.lowPriority)
	if lowPriority {
		if !info.highPriority {
			info.lowPriority = true
		}
	} else {
		info.lowPriority = false
		info.highPriority = true
	}

	if !info.pending {
		rd.addToReady(repo, info)
		if added && !rd.locked {
			rd.readyCond.Signal()
		}
	}
}

func (rd *repoDispatcher) Take() string {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	for !rd.someReady() {
		rd.readyCond.Wait()
	}

	var repo string
	var info *repoInfo
	var repos *map[string]*repoInfo

	if len(rd.highPriorityRepos) > 0 {
		repos = &rd.highPriorityRepos
	} else if len(rd.lowPriorityRepos) > 0 {
		repos = &rd.lowPriorityRepos
	}

	for repo = range *repos {
		info = (*repos)[repo]
		delete(*repos, repo)
		break
	}

	info.pending = true
	rd.pendingCount++

	info.lowPriority = false
	info.highPriority = false

	return repo
}

func (rd *repoDispatcher) Release(repo string) {
	rd.mutex.Lock()
	defer rd.mutex.Unlock()

	var info = rd.repos[repo]
	if !info.pending {
		panic("Release called for a repo that isn't pending")
	}

	info.pending = false
	rd.pendingCount--
	rd.pendingCond.Signal()

	if info.lowPriority || info.highPriority {
		rd.addToReady(repo, info)
		if !rd.locked {
			rd.readyCond.Signal()
		}
	} else {
		delete(rd.repos, repo)
	}
}

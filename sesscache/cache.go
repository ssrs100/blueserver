package sesscache

//var s = sessionCache{
//	cache: make(map[string]string),
//}
//
//type sessionCache struct {
//	sync.RWMutex
//	cache map[string]string
//}
//
//func Set(key, value string) {
//	s.Lock()
//	s.cache[key] = value
//	s.Unlock()
//}
//
//func Get(key string) string {
//	s.RLock()
//	defer s.RUnlock()
//	return s.cache[key]
//}
//
//func Del(key string) {
//	s.Lock()
//	delete(s.cache, key)
//	s.Unlock()
//}

package lib

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

const NOT_SUPPORTED = "N/S"
const NOT_AVAILABLE = "N/A"

type Stats map[string]interface{}

// NewStats returns a new stat
func NewStats() Stats {
	return make(Stats)
}

func ToStats(inMap interface{}) Stats {
	outMap := Stats{}
	if m, ok := inMap.(map[string]Stats); ok {
		for k, v := range m {
			outMap[k] = v
		}
	} else if m, ok := inMap.(map[string]map[string]Stats); ok {
		for k1, mv1 := range m {
			outMv := Stats{}
			for k2, v2 := range mv1 {
				outMv[k2] = v2
			}
			outMap[k1] = outMv
		}
	}
	return outMap
}

func ToString(sv interface{}) (string, error) {
	switch v := sv.(type) {
	case string:
		return v, nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return "", fmt.Errorf("Invalid value type")
	}
}

func (s Stats) Len() int {
	return len(s)
}

func (s Stats) Clone() Stats {
	res := make(Stats, len(s))
	for k, v := range s {
		res[k] = v
	}
	return res
}

func (s Stats) FindKeysPath(del string, keys ...string) map[string][]string {
	paths := map[string][]string{}
	for _, key := range keys {
		l := s.FindKeyPath(del, key)
		paths[key] = l
	}
	return paths
}

func (s Stats) FindKeyPath(del string, key string) []string {
	var pathList []string
	if del == "" {
		del = "/"
	}
	for skey, sval := range s {
		if key == skey {
			pathList = append(pathList, key)
		} else if v, ok := sval.(Stats); ok {
			paths := v.FindKeyPath(del, key)
			for _, path := range paths {
				p := skey + del + path
				pathList = append(pathList, p)
			}
		}
	}
	return pathList
}

// Value should be an float64 or a convertible string
// this function never panics
func (s Stats) AggregateStats(other Stats) {
	for k, v := range other {
		if val := addValues(s[k], v); val != nil {
			s[k] = val
		}
	}
}

func (s Stats) ToStringValues() map[string]interface{} {
	res := make(map[string]interface{}, len(s))
	for k, v := range s {
		sv, err := ToString(v)
		if err != nil {
			res[k] = sv
		} else {
			res[k] = v
		}
	}
	return res
}

func (s Stats) Get(name string, aliases ...string) interface{} {
	if val, exists := s[name]; exists {
		return val
	}

	for _, alias := range aliases {
		if val, exists := s[alias]; exists {
			return val
		}
	}

	return nil
}

func (s Stats) ExistsGet(name string) (interface{}, bool) {
	val, exists := s[name]
	return val, exists
}

func (s Stats) GetMulti(names ...string) Stats {
	res := make(Stats, len(names))
	for _, name := range names {
		if val, exists := s[name]; exists {
			res[name] = val
		} else {
			res[name] = NOT_AVAILABLE
		}
	}

	return res
}

func (s Stats) Del(names ...string) {
	for _, name := range names {
		delete(s, name)
	}
}

// TryInt - Value should be an int64 or a convertible string; otherwise defValue is returned
// this function never panics
func (s Stats) TryInt(name string, defValue int64, aliases ...string) int64 {
	field := s.Get(name, aliases...)
	if field != nil {
		if value, ok := field.(int64); ok {
			return value
		}
		if value, ok := field.(float64); ok {
			return int64(value)
		}
	}
	return defValue
}

// Value should be an int64, and should exist; otherwise panics
func (s Stats) Int(name string, aliases ...string) int64 {
	field := s.Get(name, aliases...)
	return field.(int64)
}

// Value should be an float64 or a convertible string; otherwise defValue is returned
// this function never panics
func (s Stats) TryFloat(name string, defValue float64, aliases ...string) float64 {
	field := s.Get(name, aliases...)
	if field != nil {
		if value, ok := field.(float64); ok {
			return value
		}
		if value, ok := field.(int64); ok {
			return float64(value)
		}
	}
	return defValue
}

// Value should be an int64 or a convertible string; otherwise defValue is returned
// this function never panics
func (s Stats) TryString(name string, defValue string, aliases ...string) string {
	field := s.Get(name, aliases...)
	if field != nil {
		if value, ok := field.(string); ok {
			return value
		}
	}
	return defValue
}

// Value should be an int64 or a convertible string; otherwise defValue is returned
// this function never panics
func (s Stats) TryStringP(name string, defValue string, aliases ...string) *string {
	field := s.Get(name, aliases...)
	if field != nil {
		s := field.(string)
		return &s
	}
	return &defValue
}

func (s Stats) TryList(name string, aliases ...string) []string {
	field := s.Get(name, aliases...)
	if field != nil {
		if value, ok := field.([]string); ok {
			return value
		}
	}
	return nil
}

func (s Stats) TryStats(name string, aliases ...string) Stats {
	field := s.Get(name, aliases...)
	if field != nil {
		if value, ok := field.(Stats); ok {
			return value
		}
	}
	return nil
}

func (s Stats) TryStatsList(name string, aliases ...string) []Stats {
	field := s.Get(name, aliases...)
	if field != nil {
		if value, ok := field.([]Stats); ok {
			return value
		}
	}
	return nil
}

func (s Stats) Flatten(sep string) Stats {
	res := make(Stats, len(s))
	for k, v := range s {
		switch v := v.(type) {
		case map[string]interface{}:
			for k2, v2 := range Stats(v).Flatten(sep) {
				res[k+sep+k2] = v2
			}
		case Stats:
			for k2, v2 := range v.Flatten(sep) {
				res[k+sep+k2] = v2
			}
			//FIXME:
		case []interface{}:
			for _, list_elem := range v {
				switch list_elem := list_elem.(type) {
				case string:
					res[k] = v
				default:
					for k2, v2 := range list_elem.(Stats).Flatten(sep) {
						res[k+sep+"["+list_elem.(Stats)["name"].(string)+"]"+sep+k2] = v2
					}
				}
			}
		case []Stats:
			for _, list_elem := range v {
				for k2, v2 := range list_elem.Flatten(sep) {
					res[k+sep+"["+list_elem["name"].(string)+"]"+sep+k2] = v2
				}
			}
		default:
			res[k] = v
		}
	}

	return res
}

// Type should be map[string]interface{} otherwise same map is returned.
// info_parser needs parsed values
func (s Stats) ToParsedValues() map[string]interface{} {
	res := make(map[string]interface{}, len(s))

	for k, val := range s {
		valStr, ok := val.(string)
		if !ok {
			res[k] = val
			continue
		}
		if value, err := strconv.ParseInt(valStr, 10, 64); err == nil {
			res[k] = value
		} else if value, err := strconv.ParseFloat(valStr, 64); err == nil {
			res[k] = value
		} else if value, err := strconv.ParseBool(valStr); err == nil {
			res[k] = value
		} else if _, err := strconv.ParseUint(valStr, 10, 64); err == nil {
			// this uint can not fit in int. uint numbers should not be put in tsdb. There may be few config/stats which
			// are initialized with biggest uint. Put may be as zero but not as string.
			res[k] = 0
		} else {
			res[k] = valStr
		}
	}

	return res
}

// GetInnerVal will give inner map from a nested map, not the base field.
// By using this we can get the map at any level. if this map is the
// lower most then there are other TryString type calls which can be
// used to further get any specific type of base field
func (s Stats) GetInnerVal(keys ...string) Stats {
	temp := s
	for _, k := range keys {
		if val, ok := temp[k]; ok {
			if temp, ok = val.(Stats); !ok {
				return nil
			}
		} else {
			return nil
		}
	}
	return temp
}

/**********************************************************************
					Type SyncStats
***********************************************************************/
type SyncStats struct {
	_Stats Stats

	mutex sync.RWMutex
}

func NewSyncStats(stats Stats) *SyncStats {
	return &SyncStats{
		_Stats: stats,
	}
}

func (s *SyncStats) SetStats(info Stats) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s._Stats = info
}

func (s *SyncStats) Set(name string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s._Stats[name] = value
}

func (s *SyncStats) Clone() Stats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s._Stats.Clone()
}

func (s *SyncStats) Exists(name string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, exists := s._Stats[name]
	return exists
}

func (s *SyncStats) CloneInto(res Stats) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for k, v := range s._Stats {
		res[k] = v
	}
}

func (s *SyncStats) Get(name string, aliases ...string) interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s._Stats.Get(name, aliases...)
}

func (s *SyncStats) ExistsGet(name string) (interface{}, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s._Stats.ExistsGet(name)
}

func (s *SyncStats) GetMulti(names ...string) Stats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s._Stats.GetMulti(names...)
}

func (s *SyncStats) Del(names ...string) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	s._Stats.Del(names...)
}

// Value MUST exist, and MUST be an int64 or a convertible string.
// Panics if the above constraints are not met
func (s *SyncStats) Int(name string, aliases ...string) int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s._Stats.Int(name, aliases...)
}

// Value should be an int64 or a convertible string; otherwise defValue is returned
// this function never panics
func (s *SyncStats) TryInt(name string, defValue int64, aliases ...string) int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s._Stats.TryInt(name, defValue, aliases...)
}

// Value should be an float64 or a convertible string; otherwise defValue is returned
// this function never panics
func (s *SyncStats) TryFloat(name string, defValue float64, aliases ...string) float64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s._Stats.TryFloat(name, defValue, aliases...)
}

// Value should be a string; otherwise defValue is returned
// this function never panics
func (s *SyncStats) TryString(name string, defValue string, aliases ...string) string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s._Stats.TryString(name, defValue, aliases...)
}

// Value should be an float64 or a convertible string
// this function never panics
func (s *SyncStats) AggregateStatsTo(other Stats) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	other.AggregateStats(s._Stats)
}

/*
	Utility functions
*/

// StatsBy is the type of a "less" function that defines the ordering of its Stats arguments.
type StatsBy func(fieldName string, p1, p2 Stats) bool

var ByFloatField = func(fieldName string, p1, p2 Stats) bool {
	return p1.TryFloat(fieldName, 0) < p2.TryFloat(fieldName, 0)
}

var ByIntField = func(fieldName string, p1, p2 Stats) bool {
	return p1.TryInt(fieldName, 0) < p2.TryInt(fieldName, 0)
}

var ByStringField = func(fieldName string, p1, p2 Stats) bool {
	return p1.TryString(fieldName, "") < p2.TryString(fieldName, "")
}

func (by StatsBy) Sort(fieldName string, statsList []Stats) {
	ps := &statsSorter{
		fieldName: fieldName,
		statsList: statsList,
		by:        by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

func (by StatsBy) SortReverse(fieldName string, statsList []Stats) {
	ps := &statsSorter{
		fieldName: fieldName,
		statsList: statsList,
		by:        by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(sort.Reverse(ps))
}

// statsSorter joins a StatsBy function and a slice of statsList to be sorted.
type statsSorter struct {
	fieldName string
	statsList []Stats
	by        func(fieldName string, p1, p2 Stats) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *statsSorter) Len() int {
	return len(s.statsList)
}

// Swap is part of sort.Interface.
func (s *statsSorter) Swap(i, j int) {
	s.statsList[i], s.statsList[j] = s.statsList[j], s.statsList[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *statsSorter) Less(i, j int) bool {
	return s.by(s.fieldName, s.statsList[i], s.statsList[j])
}

func addValues(v1, v2 interface{}) interface{} {
	// If v1 is nil then also it should copy data from v2
	// else multilayers stats will not work if v1 is nil
	if v1S, ok := v1.(Stats); ok || v1 == nil {
		if v2S, ok := v2.(Stats); ok {
			res := Stats{}
			res.AggregateStats(v1S)
			res.AggregateStats(v2S)
			return res
		}
	}

	v1Vali, v1i := v1.(int64)
	v2Vali, v2i := v2.(int64)

	v1Valf, v1f := v1.(float64)
	v2Valf, v2f := v2.(float64)

	if v1i && v2i {
		return v1Vali + v2Vali
	} else if v1f && v2f {
		return v1Valf + v2Valf
	} else if v1i && v2f {
		return float64(v1Vali) + v2Valf
	} else if v1f && v2i {
		return v1Valf + float64(v2Vali)
	} else if v2 == nil && (v1i || v1f) {
		return v1
	} else if v1 == nil && (v2i || v2f) {
		return v2
	}
	return nil
}

// Getraw(map, key ...)
// input is full qualified name
func (input Stats) GetRaw(keys ...string) interface{} {
	var d = input
	for _, k := range keys {
		// TODO: Just (Map) does not work !!!
		if d1, ok := d[k].(Stats); ok {
			d = d1
			continue
		} else {
			return d[k]
		}
	}
	return d
}

// CopyMap(map)
func (input Stats) DeepClone() Stats {
	var result = make(Stats)
	for k := range input {
		v := input[k]
		switch v := v.(type) {
		case string:
			result[k] = v
			break
		default:
			result[k] = v.(Stats).DeepClone()
			break
		}
	}
	return result
}

// GetType(map, key ...)
/*
func (input Stats) GetType(keys ...string) interface{} {
	d := input
	for _, k := range keys {
		if d1, exists := d[k].(Stats); exists {
			d = d1
			continue
		} else {
			d = d[k]
			break
		}
	}
	return d
}
*/

func ToStatsDeep(input Stats) Stats {
	result := make(Stats)

	if len(input) == 0 {
		return result
	}

	for k, v := range input {
		switch v.(type) {
		case Stats:
			m, _ := v.(Stats)
			result[k] = ToStatsDeep(m)
		case map[string]interface{}:
			m, _ := v.(map[string]interface{})
			result[k] = ToStatsDeep(m)
		case []Stats:
			l, _ := v.([]Stats)
			list := make([]Stats, 0, len(l))
			for _, i := range l {
				list = append(list, ToStatsDeep(i))
			}
			result[k] = list
		case []map[string]interface{}:
			l, _ := v.([]map[string]interface{})
			list := make([]Stats, 0, len(l))
			for _, i := range l {
				list = append(list, ToStatsDeep(i))
			}
			result[k] = list
		case []interface{}:
			l, _ := v.([]interface{})
			list := make([]interface{}, 0, len(l))
			for _, i := range l {
				switch i.(type) {
				case Stats:
					list = append(list, ToStatsDeep(i.(Stats)))
				case map[string]interface{}:
					list = append(list, ToStatsDeep(i.(map[string]interface{})))
				default:
					list = append(list, i)
				}

			}
			result[k] = list
		default:
			result[k] = v
		}
	}

	return result
}

// DeepCopy Make a deep copy from src into dst. src, dst both should be pointer
func DeepCopy(dst interface{}, src interface{}) error {
	if dst == nil {
		return fmt.Errorf("dst cannot be nil")
	}
	if src == nil {
		return fmt.Errorf("src cannot be nil")
	}
	bytes, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("Unable to marshal src: %s", err)
	}
	err = json.Unmarshal(bytes, dst)
	if err != nil {
		return fmt.Errorf("Unable to unmarshal into dst: %s", err)
	}
	return nil
}

package xerr

// 成功返回
const OK uint64 = 200

/**(前3位代表业务,后三位代表具体功能)**/

// 全局错误码
const UNKNOWN_ERROR uint64 = 100000
const SERVER_COMMON_ERROR uint64 = 100001
const REUQEST_PARAM_ERROR uint64 = 100002
const TOKEN_EXPIRE_ERROR uint64 = 100003
const TOKEN_GENERATE_ERROR uint64 = 100004
const DB_ERROR uint64 = 100005
const DB_UPDATE_AFFECTED_ZERO_ERROR uint64 = 100006

// 模块

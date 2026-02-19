import { clearSession } from "@/utils/session";

/**
 * 安全退出登录函数
 * 清除登录相关数据，但保留用户偏好设置（如主题）
 */
export const safeLogout = () => {
  clearSession();
};

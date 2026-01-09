"use client";

import { useState, useEffect } from "react";
import MainLayout from "@/components/layout/MainLayout";
import Header from "@/components/layout/Header";
import Modal from "@/components/ui/Modal";
import {
  PlusIcon,
  MagnifyingGlassIcon,
  PencilSquareIcon,
  TrashIcon,
  KeyIcon,
  ShieldCheckIcon,
  UserIcon,
} from "@heroicons/react/24/outline";
import { userApi, roleApi } from "@/lib/api/admin";
import type { User, Role } from "@/lib/api/types";

const roleConfig: Record<string, { label: string; color: string; icon: typeof ShieldCheckIcon }> = {
  admin: { label: "管理员", color: "bg-violet-500/10 text-violet-600 dark:text-violet-400 border-violet-500/20", icon: ShieldCheckIcon },
  operator: { label: "操作员", color: "bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 border-cyan-500/20", icon: KeyIcon },
  viewer: { label: "只读", color: "bg-slate-500/10 text-slate-600 dark:text-slate-400 border-slate-500/20", icon: UserIcon },
};

function getRoleDisplay(roles: Role[]) {
  if (!roles || roles.length === 0) return { label: "未分配", color: "bg-slate-500/10 text-slate-600 dark:text-slate-400 border-slate-500/20", icon: UserIcon };
  const roleName = roles[0].name;
  return roleConfig[roleName] || { label: roles[0].display_name || roleName, color: "bg-slate-500/10 text-slate-600 dark:text-slate-400 border-slate-500/20", icon: UserIcon };
}

function formatLastLogin(lastLogin?: string) {
  if (!lastLogin) return "-";
  const date = new Date(lastLogin);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "刚刚";
  if (minutes < 60) return `${minutes} 分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} 小时前`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days} 天前`;
  return date.toLocaleDateString();
}

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [saving, setSaving] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [formData, setFormData] = useState({
    username: "",
    full_name: "",
    email: "",
    password: "",
    role_id: 0,
  });

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [usersRes, rolesRes] = await Promise.all([
        userApi.list(),
        roleApi.list(),
      ]);
      if (usersRes.success) {
        setUsers(usersRes.data.users || usersRes.data.items || []);
      }
      if (rolesRes.success) {
        setRoles(rolesRes.data.roles || rolesRes.data.items || []);
      }
    } catch (error) {
      console.error("Failed to fetch users:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    setFormData({ username: "", full_name: "", email: "", password: "", role_id: roles[0]?.id || 0 });
    setShowAddModal(true);
  };

  const handleEdit = (user: User) => {
    setSelectedUser(user);
    setFormData({
      username: user.username,
      full_name: user.full_name,
      email: user.email,
      password: "",
      role_id: user.roles?.[0]?.id || 0,
    });
    setShowEditModal(true);
  };

  const handleDelete = (user: User) => {
    setSelectedUser(user);
    setShowDeleteModal(true);
  };

  const handleToggleStatus = async (user: User) => {
    try {
      const newStatus = user.status === "active" ? "disabled" : "active";
      await userApi.updateStatus(user.id, newStatus);
      setUsers(prev => prev.map(u => u.id === user.id ? { ...u, status: newStatus } : u));
    } catch (error) {
      console.error("Failed to update user status:", error);
    }
  };

  const handleSaveAdd = async () => {
    setSaving(true);
    try {
      const res = await userApi.create({
        username: formData.username,
        password: formData.password,
        email: formData.email || undefined,
        full_name: formData.full_name || undefined,
        role_ids: formData.role_id ? [formData.role_id] : undefined,
      });
      if (res.success) {
        await fetchData();
        setShowAddModal(false);
      }
    } catch (error) {
      console.error("Failed to create user:", error);
    } finally {
      setSaving(false);
    }
  };

  const handleSaveEdit = async () => {
    if (!selectedUser) return;
    setSaving(true);
    try {
      const updateData: { email?: string; full_name?: string; password?: string } = {
        email: formData.email,
        full_name: formData.full_name,
      };
      if (formData.password) {
        updateData.password = formData.password;
      }
      await userApi.update(selectedUser.id, updateData);
      if (formData.role_id && formData.role_id !== selectedUser.roles?.[0]?.id) {
        await userApi.updateRoles(selectedUser.id, [formData.role_id]);
      }
      await fetchData();
      setShowEditModal(false);
    } catch (error) {
      console.error("Failed to update user:", error);
    } finally {
      setSaving(false);
    }
  };

  const handleConfirmDelete = async () => {
    if (!selectedUser) return;
    setSaving(true);
    try {
      await userApi.delete(selectedUser.id);
      await fetchData();
      setShowDeleteModal(false);
    } catch (error) {
      console.error("Failed to delete user:", error);
    } finally {
      setSaving(false);
    }
  };

  const filteredUsers = users.filter(user => 
    user.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
    user.full_name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    user.email?.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <MainLayout>
      <Header title="用户管理" breadcrumb={["用户管理"]} />
      <div className="flex-1 overflow-y-auto p-6">
        {/* 页面头部 */}
        <div className="flex items-start justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-slate-800 dark:text-white">用户管理</h2>
            <p className="text-sm text-slate-500 mt-1">管理系统用户账号和权限</p>
          </div>
          <button
            onClick={handleAdd}
            className="flex items-center gap-2 px-4 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-600 hover:to-blue-700 rounded-xl shadow-lg shadow-cyan-500/30 transition-all"
          >
            <PlusIcon className="w-5 h-5" />
            添加用户
          </button>
        </div>

        {/* 搜索栏 */}
        <div className="flex items-center gap-4 mb-5 p-4 glass-card rounded-2xl">
          <div className="relative flex-1 max-w-md">
            <MagnifyingGlassIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="搜索用户名、昵称、邮箱..."
              className="w-full pl-10 pr-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            />
          </div>
        </div>

        {/* 用户列表 */}
        <div className="glass-card rounded-2xl overflow-hidden">
          {loading ? (
            <div className="p-8 text-center text-slate-500">加载中...</div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="bg-slate-100 dark:bg-[#0f1729]/50">
                  <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">用户</th>
                  <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">邮箱</th>
                  <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">角色</th>
                  <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">状态</th>
                  <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">最后登录</th>
                  <th className="px-5 py-4 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-white/5">
                {filteredUsers.map((user) => {
                  const roleDisplay = getRoleDisplay(user.roles);
                  const RoleIcon = roleDisplay.icon;
                  return (
                    <tr key={user.id} className="hover:bg-slate-50 dark:hover:bg-white/[0.02]">
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-full bg-gradient-to-br from-cyan-500 to-blue-600 flex items-center justify-center text-white font-medium shadow-lg shadow-cyan-500/20">
                            {(user.full_name || user.username).charAt(0).toUpperCase()}
                          </div>
                          <div>
                            <div className="text-sm font-medium text-slate-800 dark:text-slate-200">{user.full_name || user.username}</div>
                            <div className="text-xs text-slate-500">@{user.username}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-4">
                        <span className="text-sm text-slate-600 dark:text-slate-400">{user.email || "-"}</span>
                      </td>
                      <td className="px-5 py-4">
                        <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-full border ${roleDisplay.color}`}>
                          <RoleIcon className="w-3.5 h-3.5" />
                          {roleDisplay.label}
                        </span>
                      </td>
                      <td className="px-5 py-4">
                        <button
                          onClick={() => handleToggleStatus(user)}
                          className={`w-10 h-5 rounded-full transition-all relative ${
                            user.status === "active"
                              ? "bg-gradient-to-r from-emerald-500 to-green-600"
                              : "bg-slate-300 dark:bg-white/20"
                          }`}
                        >
                          <span
                            className={`absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-all ${
                              user.status === "active" ? "left-5" : "left-0.5"
                            }`}
                          />
                        </button>
                      </td>
                      <td className="px-5 py-4">
                        <span className="text-sm text-slate-500">{formatLastLogin(user.last_login)}</span>
                      </td>
                      <td className="px-5 py-4">
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() => handleEdit(user)}
                            className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 flex items-center justify-center transition-all"
                          >
                            <PencilSquareIcon className="w-4 h-4 text-slate-500 dark:text-slate-400" />
                          </button>
                          <button
                            onClick={() => handleDelete(user)}
                            disabled={user.roles?.some(r => r.name === "admin")}
                            className="w-8 h-8 rounded-lg bg-rose-100 dark:bg-rose-500/10 hover:bg-rose-200 dark:hover:bg-rose-500/20 border border-rose-200 dark:border-rose-500/20 flex items-center justify-center transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            <TrashIcon className="w-4 h-4 text-rose-500 dark:text-rose-400" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* 添加用户弹窗 */}
      <Modal isOpen={showAddModal} onClose={() => setShowAddModal(false)} title="添加用户" size="md">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">用户名 *</label>
            <input
              type="text"
              value={formData.username}
              onChange={(e) => setFormData(prev => ({ ...prev, username: e.target.value }))}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">昵称</label>
            <input
              type="text"
              value={formData.full_name}
              onChange={(e) => setFormData(prev => ({ ...prev, full_name: e.target.value }))}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">邮箱</label>
            <input
              type="email"
              value={formData.email}
              onChange={(e) => setFormData(prev => ({ ...prev, email: e.target.value }))}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">密码 *</label>
            <input
              type="password"
              value={formData.password}
              onChange={(e) => setFormData(prev => ({ ...prev, password: e.target.value }))}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">角色</label>
            <select
              value={formData.role_id}
              onChange={(e) => setFormData(prev => ({ ...prev, role_id: Number(e.target.value) }))}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            >
              {roles.map(role => (
                <option key={role.id} value={role.id}>{role.display_name || role.name}</option>
              ))}
            </select>
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <button onClick={() => setShowAddModal(false)} className="px-4 py-2.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all">
              取消
            </button>
            <button 
              onClick={handleSaveAdd} 
              disabled={!formData.username || !formData.password || saving}
              className="px-6 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 rounded-xl shadow-lg shadow-cyan-500/30 disabled:opacity-50"
            >
              {saving ? "添加中..." : "添加"}
            </button>
          </div>
        </div>
      </Modal>

      {/* 编辑用户弹窗 */}
      <Modal isOpen={showEditModal} onClose={() => setShowEditModal(false)} title="编辑用户" size="md">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">用户名</label>
            <input
              type="text"
              value={formData.username}
              disabled
              className="w-full px-4 py-2.5 text-sm bg-slate-200 dark:bg-[#0f1729]/30 border border-slate-200 dark:border-white/10 rounded-xl text-slate-500 dark:text-slate-400"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">昵称</label>
            <input
              type="text"
              value={formData.full_name}
              onChange={(e) => setFormData(prev => ({ ...prev, full_name: e.target.value }))}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">邮箱</label>
            <input
              type="email"
              value={formData.email}
              onChange={(e) => setFormData(prev => ({ ...prev, email: e.target.value }))}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">新密码（留空不修改）</label>
            <input
              type="password"
              value={formData.password}
              onChange={(e) => setFormData(prev => ({ ...prev, password: e.target.value }))}
              placeholder="留空则不修改密码"
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">角色</label>
            <select
              value={formData.role_id}
              onChange={(e) => setFormData(prev => ({ ...prev, role_id: Number(e.target.value) }))}
              className="w-full px-4 py-2.5 text-sm bg-slate-100 dark:bg-[#0f1729]/50 border border-slate-200 dark:border-white/10 rounded-xl text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-cyan-500/50"
            >
              {roles.map(role => (
                <option key={role.id} value={role.id}>{role.display_name || role.name}</option>
              ))}
            </select>
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <button onClick={() => setShowEditModal(false)} className="px-4 py-2.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all">
              取消
            </button>
            <button 
              onClick={handleSaveEdit} 
              disabled={saving}
              className="px-6 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-cyan-500 to-blue-600 rounded-xl shadow-lg shadow-cyan-500/30 disabled:opacity-50"
            >
              {saving ? "保存中..." : "保存"}
            </button>
          </div>
        </div>
      </Modal>

      {/* 删除确认弹窗 */}
      <Modal isOpen={showDeleteModal} onClose={() => setShowDeleteModal(false)} title="删除用户" size="sm">
        <div className="text-center">
          <p className="text-slate-700 dark:text-slate-300 mb-6">
            确定要删除用户 <span className="font-semibold">{selectedUser?.full_name || selectedUser?.username}</span> 吗？
          </p>
          <div className="flex justify-center gap-3">
            <button onClick={() => setShowDeleteModal(false)} className="px-6 py-2.5 text-sm font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-white/5 hover:bg-slate-200 dark:hover:bg-white/10 border border-slate-200 dark:border-white/10 rounded-xl transition-all">
              取消
            </button>
            <button 
              onClick={handleConfirmDelete} 
              disabled={saving}
              className="px-6 py-2.5 text-sm font-medium text-white bg-gradient-to-r from-rose-500 to-red-600 rounded-xl shadow-lg shadow-rose-500/30 disabled:opacity-50"
            >
              {saving ? "删除中..." : "删除"}
            </button>
          </div>
        </div>
      </Modal>
    </MainLayout>
  );
}

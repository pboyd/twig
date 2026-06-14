import { Routes, Route, Navigate } from "react-router";
import LoginPage from "./pages/LoginPage";
import TaskTreePage from "./pages/TaskTreePage";
import TaskDetailPage from "./pages/TaskDetailPage";
import PlanPage from "./pages/PlanPage";
import DownloadPage from "./pages/DownloadPage";
import AccountPage from "./pages/AccountPage";
import { ToastProvider } from "./context/ToastProvider";

export default function App() {
  return (
    <ToastProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/tasks" element={<TaskTreePage />} />
        <Route path="/tasks/:id" element={<TaskDetailPage />} />
        <Route path="/plan" element={<PlanPage />} />
        <Route path="/download" element={<DownloadPage />} />
        <Route path="/account" element={<AccountPage />} />
        <Route path="*" element={<Navigate to="/tasks" replace />} />
      </Routes>
    </ToastProvider>
  );
}

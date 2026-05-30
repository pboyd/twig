import { Routes, Route, Navigate } from "react-router";
import LoginPage from "./pages/LoginPage";
import TaskTreePage from "./pages/TaskTreePage";
import TaskDetailPage from "./pages/TaskDetailPage";

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/tasks" element={<TaskTreePage />} />
      <Route path="/tasks/:id" element={<TaskDetailPage />} />
      <Route path="*" element={<Navigate to="/tasks" replace />} />
    </Routes>
  );
}

import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { VaultProvider } from './context/VaultContext';
import ProtectedRoute from './components/ProtectedRoute';
import LandingPage from './pages/LandingPage';
import PricingPage from './pages/PricingPage';
import AboutPage from './pages/AboutPage';
import LoginPage from './pages/LoginPage';
import NotesPage from './pages/NotesPage';
import AccountPage from './pages/AccountPage';
import ImportPage from './pages/ImportPage';
import NotFoundPage from './pages/NotFoundPage';

export default function App() {
  return (
    <VaultProvider>
      <BrowserRouter>
      <Routes>
        <Route path="/" element={<LandingPage />} />
        <Route path="/pricing" element={<PricingPage />} />
        <Route path="/about" element={<AboutPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/dashboard" element={<Navigate to="/notes" replace />} />
        <Route
          path="/notes"
          element={
            <ProtectedRoute>
              <NotesPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/account"
          element={
            <ProtectedRoute>
              <AccountPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/import"
          element={
            <ProtectedRoute>
              <ImportPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/dev/import"
          element={<Navigate to="/import" replace />}
        />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
      </BrowserRouter>
    </VaultProvider>
  );
}

import React from 'react';
import { Routes, Route } from 'react-router-dom';
import MainLayout from './components/layout/MainLayout';
import HomePage from './pages/HomePage';
// Import other pages here as they are created
// import NotFoundPage from './pages/NotFoundPage';

function App() {
  return (
    <MainLayout>
      <Routes>
        <Route path="/" element={<HomePage />} />
        {/* Add other routes here */}
        {/* <Route path="*" element={<NotFoundPage />} /> */}
      </Routes>
    </MainLayout>
  );
}

export default App;

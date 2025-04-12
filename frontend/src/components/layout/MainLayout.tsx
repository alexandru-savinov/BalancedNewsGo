import React, { ReactNode } from 'react';
import Header from './Header';

interface MainLayoutProps {
  children: ReactNode;
}

const MainLayout: React.FC<MainLayoutProps> = ({ children }) => {
  return (
    <div>
      <Header />
      <main style={{ padding: '1rem' }}>
        {children}
      </main>
      {/* Footer could be added here later */}
    </div>
  );
};

export default MainLayout;
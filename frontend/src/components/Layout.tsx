import { Outlet } from 'react-router-dom';
import { Footer } from './Footer';
import { NavBar } from './NavBar';

export function Layout() {
  return (
    <div className="app-shell">
      <NavBar />
      <main className="app-main">
        <div className="container">
          <Outlet />
        </div>
      </main>
      <Footer />
    </div>
  );
}

import React from 'react';
import { NavLink } from 'react-router-dom';
import { 
  HomeIcon, 
  ChatBubbleLeftIcon, 
  CpuChipIcon,
  Squares2X2Icon
} from '@heroicons/react/24/outline';

const Navigation: React.FC = () => {
  const navItems = [
    { path: '/', label: 'Dashboard', icon: HomeIcon, ariaLabel: 'dashboard' },
    { path: '/graph', label: 'Graph View', icon: Squares2X2Icon, ariaLabel: 'graph' },
    { path: '/chat', label: 'Chat', icon: ChatBubbleLeftIcon, ariaLabel: 'chat' },
    { path: '/agents', label: 'Agents', icon: CpuChipIcon, ariaLabel: 'agents' },
  ];

  return (
    <nav className="bg-white border-b border-gray-200" role="navigation">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex space-x-8">
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                `flex items-center space-x-2 py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                  isActive
                    ? 'border-primary-500 text-primary-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`
              }
              aria-label={item.ariaLabel}
            >
              <item.icon className="h-5 w-5" />
              <span>{item.label}</span>
            </NavLink>
          ))}
        </div>
      </div>
    </nav>
  );
};

export default Navigation;

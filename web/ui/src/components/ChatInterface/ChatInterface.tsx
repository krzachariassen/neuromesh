import React, { useState } from 'react';
import { PaperAirplaneIcon } from '@heroicons/react/24/outline';
import type { ChatMessage } from '../../types/ui';

const ChatInterface: React.FC = () => {
  const [message, setMessage] = useState('');
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      id: 'msg-1',
      content: 'Hello! I\'m your NeuroMesh AI assistant. How can I help you orchestrate your AI agents today?',
      role: 'assistant',
      timestamp: new Date(Date.now() - 60000),
    },
  ]);

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();
    if (!message.trim()) return;

    const newMessage: ChatMessage = {
      id: `msg-${Date.now()}`,
      content: message,
      role: 'user',
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, newMessage]);
    setMessage('');

    // Simulate AI response
    setTimeout(() => {
      const aiResponse: ChatMessage = {
        id: `msg-${Date.now() + 1}`,
        content: 'I understand your request. Let me coordinate with the appropriate agents to process this for you.',
        role: 'assistant',
        timestamp: new Date(),
      };
      setMessages(prev => [...prev, aiResponse]);
    }, 1000);
  };

  const formatTime = (timestamp: Date) => {
    return timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  return (
    <div data-testid="chat-interface" className="flex flex-col h-full">
      <div className="flex-shrink-0 border-b border-gray-200 px-6 py-4">
        <h1 className="text-3xl font-bold text-gray-900">Chat Interface</h1>
        <p className="text-sm text-gray-600 mt-1">Communicate with your AI agents</p>
      </div>

      {/* Messages Container */}
      <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4">
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-xs lg:max-w-md px-4 py-2 rounded-lg ${
                msg.role === 'user'
                  ? 'bg-primary-600 text-white'
                  : 'bg-gray-100 text-gray-900'
              }`}
            >
              <p className="text-sm">{msg.content}</p>
              <p
                className={`text-xs mt-1 ${
                  msg.role === 'user' ? 'text-primary-100' : 'text-gray-500'
                }`}
              >
                {formatTime(msg.timestamp)}
              </p>
            </div>
          </div>
        ))}
      </div>

      {/* Message Input */}
      <div className="flex-shrink-0 border-t border-gray-200 px-6 py-4">
        <form onSubmit={handleSendMessage} className="flex space-x-4">
          <div className="flex-1">
            <input
              type="text"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Type your message to the AI orchestrator..."
              className="w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
            />
          </div>
          <button
            type="submit"
            disabled={!message.trim()}
            className="btn-primary inline-flex items-center space-x-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <PaperAirplaneIcon className="h-4 w-4" />
            <span>Send</span>
          </button>
        </form>
      </div>
    </div>
  );
};

export default ChatInterface;

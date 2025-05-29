// api.js - Place this file in your ui/src folder
const API_BASE_URL = "http://localhost:8080/api";

// Helper function to get auth headers
const getAuthHeaders = () => {
  const token = localStorage.getItem("token");
  return {
    "Content-Type": "application/json",
    ...(token && { "Authorization": `Bearer ${token}` }),
  };
};

// Helper function to handle API responses
const handleResponse = async (response) => {
  if (!response.ok) {
    const errorData = await response.text();
    throw new Error(errorData || `HTTP error! status: ${response.status}`);
  }

  // Handle empty responses (like DELETE)
  if (response.status === 204) {
    return null;
  }

  return await response.json();
};

// Authentication API
export const authAPI = {
  // Login user
  login: async (email, password) => {
    const response = await fetch(`${API_BASE_URL}/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ email, password }),
    });

    const data = await handleResponse(response);

    // Store token and user data
    if (data.token) {
      localStorage.setItem("token", data.token);
      localStorage.setItem("user", JSON.stringify(data.user));
    }

    return data;
  },

  // Sign up new user
  signup: async (name, email, password) => {
    const response = await fetch(`${API_BASE_URL}/signup`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ name, email, password }),
    });

    const data = await handleResponse(response);

    // Store token and user data
    if (data.token) {
      localStorage.setItem("token", data.token);
      localStorage.setItem("user", JSON.stringify(data.user));
    }

    return data;
  },

  // Logout user
  logout: () => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
  },

  // Get current user from localStorage
  getCurrentUser: () => {
    const userStr = localStorage.getItem("user");
    return userStr ? JSON.parse(userStr) : null;
  },

  // Check if user is authenticated
  isAuthenticated: () => {
    return !!localStorage.getItem("token");
  },
};

// Entries API
export const entriesAPI = {
  // Get all entries for current user
  getEntries: async () => {
    const response = await fetch(`${API_BASE_URL}/entries`, {
      headers: getAuthHeaders(),
    });

    return await handleResponse(response);
  },

  // Create new entry
  createEntry: async (title, text, date = null) => {
    const entryData = {
      title,
      text,
      ...(date && { date }),
    };

    const response = await fetch(`${API_BASE_URL}/entries`, {
      method: "POST",
      headers: getAuthHeaders(),
      body: JSON.stringify(entryData),
    });

    return await handleResponse(response);
  },

  // Update existing entry
  updateEntry: async (id, title, text) => {
    const response = await fetch(`${API_BASE_URL}/entries/${id}`, {
      method: "PUT",
      headers: getAuthHeaders(),
      body: JSON.stringify({ title, text }),
    });

    return await handleResponse(response);
  },

  // Delete entry
  deleteEntry: async (id) => {
    const response = await fetch(`${API_BASE_URL}/entries/${id}`, {
      method: "DELETE",
      headers: getAuthHeaders(),
    });

    return await handleResponse(response);
  },
};

// Error handling utility
export const handleAPIError = (error) => {
  console.error("API Error:", error);

  // Handle specific error cases
  if (error.message.includes("401") || error.message.includes("Unauthorized")) {
    // Token expired or invalid
    authAPI.logout();
    window.location.href = "/"; // Redirect to login
    return "Session expired. Please login again.";
  }

  if (error.message.includes("403")) {
    return "You are not authorized to perform this action.";
  }

  if (error.message.includes("404")) {
    return "The requested resource was not found.";
  }

  if (error.message.includes("500")) {
    return "Server error. Please try again later.";
  }

  return error.message || "An unexpected error occurred.";
};

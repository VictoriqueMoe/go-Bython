config = {
  "database": {
    "host": "localhost",
    "port": 5432,
    "credentials": {
      "username": "admin",
      "password": "secret"
    }
  },
  "api": {
    "endpoints": ["/users", "/posts", "/comments"],
    "rate_limit": 1000
  }
};

def process_config(cfg) {
  for key, value in cfg.items() {
    if isinstance(value, dict) {
      print(f"{key}:");
      process_config(value);
    } else {
      print(f"  {key}: {value}");
    }
  }
}

if __name__ == "__main__" {
  print("Configuration:");
  process_config(config);

  user_data = {
    "id": 1,
    "name": "Alice",
    "roles": ["admin", "user"]
  };

  print(f"\nUser: {user_data['name']}");
  print(f"Roles: {', '.join(user_data['roles'])}");
}
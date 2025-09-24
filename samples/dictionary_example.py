class UserManager {
    def __init__(self) {
        self.users = {};
        self.next_id = 1;
    }

    def add_user(self, name, email) {
        user = {"id": self.next_id, "name": name, "email": email, "active": True};
        self.users[self.next_id] = user;
        self.next_id += 1;
        return user;
    }

    def get_user(self, user_id) {
        return self.users.get(user_id);
    }

    def update_user(self, user_id, **kwargs) {
        if user_id in self.users {
            for key, value in kwargs.items() {
                self.users[user_id][key] = value;
            }
            return self.users[user_id];
        }
        return None;
    }

    def list_active_users(self) {
        active = [];
        for user_id, user in self.users.items() {
            if user["active"] {
                active.append(user);
            }
        }
        return active;
    }
}

if __name__ == "__main__" {
    manager = UserManager();

    user1 = manager.add_user("Alice", "alice@example.com");
    user2 = manager.add_user("Bob", "bob@example.com");
    user3 = manager.add_user("Charlie", "charlie@example.com");

    print(f"Created users: {manager.users}");

    manager.update_user(2, active=False);
    print(f"\nActive users:");
    for user in manager.list_active_users() {
        print(f"  - {user['name']} ({user['email']})");
    }

    config = {"app_name": "UserManager", "version": "1.0", "settings": {"debug": True, "max_users": 100}};

    print(f"\nConfig: {config}");
    print(f"Debug mode: {config['settings']['debug']}");
}
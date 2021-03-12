INSERT INTO `money_manager`.`wallet` (`user_id`, `balance`)
VALUES  ('1', '100'),
        ('2', '100'),
        ('3', '100'),
        ('4', '100');

INSERT INTO `money_manager`.`friendship` (`user_one_id`, `user_two_id`, `status`, `action_user_id`)
VALUES  ('1', '2', 'pending', '2'),
        ('1', '3', 'pending', '1'),
        ('2', '3', 'accepted', '3'),
        ('1', '4', 'accepted', '4');


INSERT INTO `money_manager`.`categories` (`id`, `c_type`, `name`)
VALUES  ('1', 'expense', 'loan'),
        ('2', 'expense', 'repay'),
        ('3', 'expense', 'food'),
        ('4', 'expense', 'home'),
        ('5', 'expense', 'car'),
        ('6', 'income', 'debt'),
        ('7', 'income', 'receive'),
        ('8', 'income', 'salary'),
        ('9', 'income', 'savings'),
        ('10', 'income', 'lottery');


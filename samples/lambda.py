# lambda functions
add = lambda x, y { x + y }
print(add(5, 3))

numbers = [1, 2, 3, 4]
squared_numbers = list(map(lambda x  {x * x}, numbers))
print(squared_numbers)

even_numbers = list(filter(lambda x { x % 2 == 0 }, numbers))
print(even_numbers)
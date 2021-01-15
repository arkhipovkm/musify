if __name__ == "__main__":
    with open('captcha.b64', 'r') as f:
        cont = f.read()
        bytes_list = [int(x) for x in cont.split(' ')]
    with open('captcha.jpg', 'wb') as f:
        f.write(bytes(bytes_list))
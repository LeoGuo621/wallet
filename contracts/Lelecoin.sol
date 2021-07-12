pragma solidity ^0.6.0;

import "./SafeMath.sol";
import "./IERC20.sol";

contract Lelecoin is IERC20 {
    using SafeMath for uint256;
    mapping(address => uint256) private _balances;
    mapping(address => mapping(address => uint256)) private _allowed;
    uint256 private _totalSupply;
    address private _admin;

    string public symbol;
    constructor() public{
        symbol = "Lelecoin";
        _admin = msg.sender;
    }

    function totalSupply() override public view returns (uint256) {
        return _totalSupply;
    }

    function balanceOf(address owner) override public view returns (uint256) {
        return _balances[owner];
    }

    function approve(address spender, uint256 value) override public returns (bool) {
        require(spender != address(0));
        _allowed[msg.sender][spender] = value;
        emit Approval(msg.sender, spender, value);
        return true;
    }

    function allowance(address owner, address spender) override public view returns (uint256) {
        return _allowed[owner][spender];
    }

    function transferFrom(
        address from,
        address to,
        uint256 value
    )
    override
    public
    returns (bool)
    {
        require(value <= _balances[from]);
        require(value <= _allowed[from][msg.sender]);
        require(to != address(0));
        _balances[from] = _balances[from].sub(value);
        _balances[to] = _balances[to].add(value);
        _allowed[from][msg.sender] = _allowed[from][msg.sender].sub(value);
        emit Transfer(from, to, value);
        return true;
    }

    function transfer(address to, uint256 value) override public returns (bool) {
        require(value <= _balances[msg.sender]);
        require(to != address(0));
        _balances[msg.sender] = _balances[msg.sender].sub(value);
        _balances[to] = _balances[to].add(value);
        emit Transfer(msg.sender, to, value);
        return true;
    }

    function mint(address to, uint256 value) public {
        require(msg.sender == _admin);
        _totalSupply = _totalSupply.add(value);
        _balances[to] = _balances[to].add(value);
        emit Transfer(address(0), to, value);
    }
}